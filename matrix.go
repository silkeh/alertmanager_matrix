package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"git.slxh.eu/prometheus/alertmanager_matrix/alertmanager"
	"git.slxh.eu/prometheus/alertmanager_matrix/matrix"
	"github.com/prometheus/alertmanager/types"
)

const (
	helpMessage = "Usage: !alert <subcommand> [options]\n\n" +
		"Available subcommands are:\n\n" +
		"- `help`: shows this message\n" +
		"- `list`: shows active alerts\n" +
		"- `list all`: shows active and silenced alerts\n" +
		"- `silence`: shows active silences\n" +
		"- `silence pending`: shows pending silences\n" +
		"- `silence expired`: shows expired silences\n" +
		"- `silence add <duration> <matchers>`: create a new silence\n" +
		"- `silence del <ids>`: create a new silence\n"
)

var client *matrix.Client
var am *alertmanager.Client

// Start the Alertmanager client
func startAlertmanagerClient(url string) (err error) {
	am, err = alertmanager.NewClient(url)
	return
}

// Start the Matrix client
func startMatrixClient(homeserver, userID, token, messageType string) (err error) {
	// Create a new client
	client, err = matrix.NewClient(homeserver, userID, token, messageType)
	if err != nil {
		return
	}

	// Register sync/message handler
	client.Syncer.OnEventType("m.room.message", messageHandler)

	// Start syncing
	go sync()

	return
}

// sync runs a never ending Matrix sync
func sync() {
	for {
		err := client.Sync()
		if err != nil {
			log.Printf("Sync error: %s", err)
		}
	}
}

// shortCommand returns the n letter abbreviation of a command.
func shortCommand(cmd []string, n int) (str string) {
	for _, c := range cmd {
		if len(c) > 0 {
			str += string(c[0])
		}
		if len(str) == n {
			break
		}
	}
	return
}

// messageHandler handles an incoming event
func messageHandler(e *matrix.Event) {
	var plain, html string
	var err error

	// Get message text
	text, ok := e.Body()
	if !ok ||
		e.Sender == client.UserID ||
		!strings.HasPrefix(text, "!alert") {
		return
	}

	// Room to send response to
	room := client.NewRoom(e.RoomID)

	// Get command
	cmd := strings.Split(text, " ")

	// Execute the action that matches the short form of the command
	switch shortCommand(cmd[1:], 2) {
	case "l":
		plain, html = Alerts(false)
	case "la":
		plain, html = Alerts(true)
	case "s":
		plain = Silences("active")
	case "sp":
		plain = Silences("pending")
	case "se":
		plain = Silences("expired")
	case "sa":
		plain = NewSilence(e.Sender, cmd[3:])
	case "sd":
		plain = DelSilence(cmd[3:])
	default:
		plain = helpMessage
	}

	// Send a Markdown message if no HTML is provided
	if html == "" {
		_, err = room.SendMarkdown(plain)
	} else {
		_, err = room.SendHTML(plain, html)
	}

	if err != nil {
		log.Print("Error: ", err)
	}
}

// Alerts returns all or non-silenced alerts
func Alerts(silenced bool) (string, string) {
	alerts, err := am.Alert.List(context.TODO(), "", silenced, false)
	if err != nil {
		return err.Error(), ""
	}
	if len(alerts) == 0 {
		return "No alerts", ""
	}

	// Map alerts to compatible type
	as := make([]*Alert, len(alerts))
	for i, a := range alerts {
		as[i] = &Alert{
			Alert:  a.Alert,
			Status: string(a.Status.State),
		}
	}

	return formatAlerts(as)
}

// Silences returns a Markdown formatted message containing silences with the specified state
func Silences(state string) string {
	silences, err := am.Silence.List(context.TODO(), "")
	if err != nil {
		return err.Error()
	}

	md := formatSilences(silences, state)

	if md == "" {
		return fmt.Sprintf("No %s silences", state)
	}

	return md
}

// NewSilence creates a new silence and returns the ID
func NewSilence(author string, args []string) string {
	if len(args) < 2 {
		return "Insufficent arguments"
	}

	matchers := args[1:]
	duration, err := parseDuration(args[0])
	if err != nil {
		return err.Error()
	}

	silence := types.Silence{
		Matchers:  make(types.Matchers, len(matchers)),
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(duration),
		CreatedBy: author,
		Comment:   "Created from Matrix",
	}

	for i, m := range matchers {
		ms := regexp.MustCompile(`(.*)=(~?)"(.*)"`).FindStringSubmatch(m)
		if ms == nil {
			return "Invalid matcher: " + m
		}
		silence.Matchers[i] = &types.Matcher{
			Name:    ms[1],
			Value:   ms[3],
			IsRegex: ms[2] == "~",
		}
	}

	id, err := am.Silence.Set(context.TODO(), silence)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("Silence created with ID *%s*", id)
}

// DelSilence deletes silences
func DelSilence(ids []string) string {
	if len(ids) == 0 {
		return "No silence IDs provided"
	}

	var errors []string

	for _, id := range ids {
		err := am.Silence.Expire(context.TODO(), id)
		if err != nil {
			errors = append(errors,
				fmt.Sprintf("Error deleting %s: %s", id, err))
		}
	}

	if errors != nil {
		return strings.Join(errors, "\n\n")
	}

	return fmt.Sprintf(
		"Silences deleted: *%s*",
		strings.Join(ids, ", "))
}
