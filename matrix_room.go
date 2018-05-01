package main

import (
	"context"
	"fmt"
	matrix "github.com/matrix-org/gomatrix"
	"github.com/prometheus/alertmanager/types"
	"regexp"
	"strings"
	"time"
)

type Room struct {
	ID string
	*matrix.Client
}

// Send a Markdown formatted message
func (r *Room) sendMarkdownMessage(md string) error {
	return r.sendMessage(md, markdown(md))
}

// Send a plain message
func (r *Room) sendText(plain string) error {
	_, err := r.SendMessageEvent(r.ID, "m.room.message",
		&MatrixMessage{
			MsgType: "m.notice",
			Body:    plain,
		},
	)
	return err
}

// Send a formatted message to a room.
func (r *Room) sendMessage(plain, html string) error {
	_, err := r.SendMessageEvent(r.ID, "m.room.message",
		&MatrixMessage{
			MsgType:       "m.notice",
			Format:        "org.matrix.custom.html",
			Body:          plain,
			FormattedBody: html,
		},
	)
	return err
}

// sendAlerts sends alerts to a room
func (r *Room) sendAlerts(silenced bool) error {
	alerts, err := am.alert.List(context.TODO(), "", silenced, false)
	if err != nil {
		return r.sendText(err.Error())
	}
	if len(alerts) == 0 {
		return r.sendText("No alerts")
	}

	// Map alerts to compatible type
	as := make([]*Alert, len(alerts))
	for i, a := range alerts {
		as[i] = &Alert{
			Alert:  a.Alert,
			Status: string(a.Status.State),
		}
	}

	plain, html := formatAlerts(as)
	return r.sendMessage(plain, html)
}

// sendSilences sends silences to a room
func (r *Room) sendSilences(state string) error {
	silences, err := am.silence.List(context.TODO(), "")
	if err != nil {
		return r.sendText(err.Error())
	}

	plain, html := formatSilences(silences, state)

	if plain == "" {
		return r.sendText(fmt.Sprintf("No %s silences", state))
	}

	return r.sendMessage(plain, html)
}

// sendNewSilence creates a new silence and sends the ID
func (r *Room) sendNewSilence(author string, args []string) error {
	if len(args) < 2 {
		return r.sendText("Insufficent arguments")
	}

	matchers := args[1:]
	duration, err := parseDuration(args[0])
	if err != nil {
		return r.sendText(err.Error())
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
			return r.sendText("Invalid matcher: " + m)
		}
		silence.Matchers[i] = &types.Matcher{
			Name:    ms[1],
			Value:   ms[3],
			IsRegex: ms[2] == "~",
		}
	}

	id, err := am.silence.Set(context.TODO(), silence)
	if err != nil {
		return r.sendText(err.Error())
	}

	return r.sendMarkdownMessage(fmt.Sprintf("Silence created with ID *%s*", id))
}

// sendDelSilence deletes one or more silences
func (r *Room) sendDelSilence(ids []string) error {
	if len(ids) == 0 {
		return r.sendText("No silence IDs provided")
	}

	for _, id := range ids {
		err := am.silence.Expire(context.TODO(), id)
		if err != nil {
			err = r.sendText(err.Error())
			if err != nil {
				return err
			}
		}
	}

	return r.sendMarkdownMessage(fmt.Sprintf(
		"Silences deleted: *%s*",
		strings.Join(ids, ", ")))
}
