package bot

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
	command     = "!alert"
	helpMessage = "Usage: !alert <subcommand> [options]\n\n" +
		"Available subcommands are:\n\n" +
		"- `help`: shows this message\n" +
		"- `list`: shows active alerts\n" +
		"- `list all`: shows active and silenced alerts\n" +
		"- `list labels`: shows labels of active alerts\n" +
		"- `silence`: shows active silences\n" +
		"- `silence pending`: shows pending silences\n" +
		"- `silence expired`: shows expired silences\n" +
		"- `silence add <duration> <matchers>`: create a new silence\n" +
		"- `silence del <ids>`: create a new silence\n"
)

// Client represents an Alertmanager/Matrix client
type Client struct {
	Matrix       *matrix.Client
	Alertmanager *alertmanager.Client
	roomList     []string
	rooms        map[string]struct{}
}

// NewClient creates and starts a new Alertmanager/Matrix client
func NewClient(homeserver, userID, token, messageType, rooms, alertmanagerURL string) (client *Client, err error) {
	client = new(Client)

	// Create Alertmanager client
	client.Alertmanager, err = alertmanager.NewClient(alertmanagerURL)
	if err != nil {
		return
	}

	// Create Matrix client
	client.Matrix, err = matrix.NewClient(homeserver, userID, token, messageType)
	if err != nil {
		return
	}

	// Register sync/message handler
	client.Matrix.Syncer.OnEventType("m.room.message", client.messageHandler)

	// Create room list
	if rooms != "" {
		client.roomList = strings.Split(rooms, ",")
	}

	return
}

// Run the client in a blocking thread
func (c *Client) Run() error {
	// Join rooms
	if c.roomList != nil {
		err := c.joinRooms(c.roomList)
		if err != nil {
			return err
		}
	}

	// Start syncing
	return c.sync()
}

// joinRooms joins a list of room IDs or aliases
func (c *Client) joinRooms(roomList []string) error {
	c.rooms = make(map[string]struct{})

	// Join all rooms
	for _, r := range roomList {
		j, err := c.Matrix.JoinRoom(r, "", nil)
		if err != nil {
			return err
		}
		c.rooms[j.RoomID] = struct{}{}
	}

	return nil
}

// sync runs a never ending Matrix sync
func (c *Client) sync() error {
	for {
		err := c.Matrix.Sync()
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

// messageHandler handles an incoming event
func (c *Client) messageHandler(e *matrix.Event) {
	var plain, html string
	var err error

	// Get message text
	text, ok := e.Body()

	// Ignore message if:
	// - no body
	// - sent by the bot itself
	// - does not start with command
	if !ok ||
		e.Sender == c.Matrix.UserID ||
		!strings.HasPrefix(text, command) {
		return
	}

	// Ignore rooms that are not explicitly allowed when this is configured
	if c.rooms != nil {
		if _, ok := c.rooms[e.RoomID]; !ok {
			log.Printf("Ignoring command from non configured room %s: %s", e.RoomID, text)
			return
		}
	}

	// Room to send response to
	room := c.Matrix.NewRoom(e.RoomID)

	// Get command
	cmd := strings.Split(text, " ")

	// Execute the action that matches the short form of the command
	switch shortCommand(cmd[1:], 2) {
	case "l":
		plain, html = c.Alerts(false, false)
	case "la":
		plain, html = c.Alerts(true, false)
	case "ll":
		plain, html = c.Alerts(true, true)
	case "s":
		plain = c.Silences("active")
	case "sp":
		plain = c.Silences("pending")
	case "se":
		plain = c.Silences("expired")
	case "sa":
		plain = c.NewSilence(e.Sender, cmd[3:])
	case "sd":
		plain = c.DelSilence(cmd[3:])
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
func (c *Client) Alerts(silenced bool, labels bool) (string, string) {
	alerts, err := c.Alertmanager.GetAlerts(silenced)
	if err != nil {
		return err.Error(), ""
	}
	if len(alerts) == 0 {
		return "No alerts", ""
	}

	return FormatAlerts(alerts, labels)
}

// Silences returns a Markdown formatted message containing silences with the specified state
func (c *Client) Silences(state string) string {
	silences, err := c.Alertmanager.Silence.List(context.TODO(), "")
	if err != nil {
		return err.Error()
	}

	md := FormatSilences(silences, state)

	if md == "" {
		return fmt.Sprintf("No %s silences", state)
	}

	return md
}

// NewSilence creates a new silence and returns the ID
func (c *Client) NewSilence(author string, args []string) string {
	if len(args) < 2 {
		return "Insufficient arguments"
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

	// Check if an ID is given instead of matchers
	if len(matchers) == 1 && !strings.ContainsRune(matchers[0], '=') {
		alert, err := c.Alertmanager.GetAlert(matchers[0])
		if err != nil {
			return err.Error()
		}
		if alert == nil {
			return fmt.Sprintf("No alert with fingerprint %s", matchers[0])
		}

		silence.Matchers = make(types.Matchers, 0, len(alert.Labels))
		for name, value := range alert.Labels {
			silence.Matchers = append(silence.Matchers, &types.Matcher{
				Name:  string(name),
				Value: string(value),
			})
		}
	} else {
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
	}

	id, err := c.Alertmanager.Silence.Set(context.TODO(), silence)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("Silence created with ID *%s*", id)
}

// DelSilence deletes silences
func (c *Client) DelSilence(ids []string) string {
	if len(ids) == 0 {
		return "No silence IDs provided"
	}

	var errors []string

	for _, id := range ids {
		err := c.Alertmanager.Silence.Expire(context.TODO(), id)
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
