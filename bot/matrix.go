// Package bot contains a Matrix Alertmanager bot.
package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/alertmanager/types"
	bot "gitlab.com/silkeh/matrix-bot"

	"github.com/silkeh/alertmanager_matrix/alertmanager"
)

var errNilClientConfig = errors.New("client config cannot be nil")

// ClientConfig contains the configuration for the client.
type ClientConfig struct {
	Homeserver      string // Matrix homeserver URL.
	UserID          string // Matrix user ID.
	Token           string // Matrix token.
	MessageType     string // Matrix NewMessage type (optional).
	Rooms           string // Comma-separated list of matrix rooms (optional).
	AlertManagerURL string // URL to the Alert Manager API.
}

// Client represents an Alertmanager/Matrix client.
type Client struct {
	Matrix       *bot.Client
	Alertmanager *alertmanager.Client
	Formatter    *Formatter
}

// NewClient creates and starts a new Alertmanager/Matrix client.
func NewClient(config *ClientConfig, formatter *Formatter) (client *Client, err error) {
	if config == nil {
		return nil, errNilClientConfig
	}

	client = &Client{
		Formatter: formatter,
	}

	// Ensure a formatter is set
	if client.Formatter == nil {
		client.Formatter = NewFormatter("", "", nil, nil)
	}

	// Create Alertmanager client
	client.Alertmanager, err = alertmanager.NewClient(config.AlertManagerURL)
	if err != nil {
		return
	}

	// Matrix bot config
	matrixConfig := &bot.ClientConfig{
		MessageType:      config.MessageType,
		CommandPrefixes:  []string{"!alert", "!alertmanager"},
		IgnoreHighlights: false,
	}

	// Create Matrix client
	client.Matrix, err = bot.NewClient(config.Homeserver, config.UserID, config.Token, matrixConfig)
	if err != nil {
		return
	}

	// Create room list
	if config.Rooms != "" {
		matrixConfig.AllowedRooms = strings.Split(config.Rooms, ",")
	}

	// Register commands
	client.Matrix.SetCommand("", client.listOnlyCommand())
	client.Matrix.SetCommand("list", client.listCommand())
	client.Matrix.SetCommand("silence", client.silenceCommand())

	return
}

// mainCommand returns the `alert` bot command.
func (c *Client) listOnlyCommand() *bot.Command {
	return &bot.Command{
		Summary: "Show active alerts.",
		MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
			return c.Alerts(false, false)
		},
	}
}

// listCommand returns the `list` bot command.
func (c *Client) listCommand() *bot.Command {
	cmd := c.listOnlyCommand()
	cmd.Subcommands = map[string]*bot.Command{
		"all": {
			Summary: "Show active and silenced alerts.",
			MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
				return c.Alerts(true, false)
			},
			Subcommands: map[string]*bot.Command{
				"labels": {
					Summary: "Shows label of active and silenced alerts.",
					MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
						return c.Alerts(true, true)
					},
				},
			},
		},
		"labels": {
			Summary: "Show labels of active alerts.",
			MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
				return c.Alerts(false, true)
			},
		},
	}

	return cmd
}

// silenceCommand returns the `silence` command.
func (c *Client) silenceCommand() *bot.Command {
	return &bot.Command{
		Summary: "Show active silences.",
		MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
			return bot.NewMarkdownMessage(c.Silences("active"))
		},
		Subcommands: map[string]*bot.Command{
			"pending": {
				Summary: "Show pending silences.",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					return bot.NewMarkdownMessage(c.Silences("pending"))
				},
			},
			"expired": {
				Summary: "Shows expired silences.",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					return bot.NewMarkdownMessage(c.Silences("expired"))
				},
			},
			"add": {
				Summary: "Create a silence.",
				Description: "Create a silence	using a `duration` and `matcher` or `fingerprint`.\n\n" +
					"A matcher matches job labels, for example: \n" +
					"```\nsilence add 1h job=\"test\" target=~\"test.*\"\n```\n" +
					"Alternative, an alert fingerprint can be given to match all labels of that alert, for example:\n" +
					"```\nsilence add 1h 04e45af092081699\n```\n",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					if len(args) <= 1 {
						return bot.NewTextMessage("Insufficient arguments.")
					}

					return bot.NewMarkdownMessage(c.NewSilence(sender, args[0], strings.Join(args[1:], " ")))
				},
			},
			"del": {
				Summary: "Delete a silence by ID.",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					return bot.NewMarkdownMessage(c.DelSilence(args))
				},
			},
		},
	}
}

// Run the client in a blocking thread.
func (c *Client) Run() error {
	err := c.joinRooms(c.Matrix.Config.AllowedRooms)
	if err != nil {
		return err
	}

	err = c.Matrix.Run()
	if err != nil {
		return fmt.Errorf("matrix error: %w", err)
	}

	return nil
}

// joinRooms joins a list of room IDs or aliases.
func (c *Client) joinRooms(roomList []string) error {
	for _, r := range roomList {
		err := c.Matrix.NewRoom(r).Join()
		if err != nil {
			return fmt.Errorf("cannot join room: %w", err)
		}
	}

	return nil
}

// Alerts returns all or non-silenced alerts.
func (c *Client) Alerts(silenced bool, labels bool) *bot.Message {
	alerts, err := c.Alertmanager.GetAlerts(silenced)
	if err != nil {
		return bot.NewTextMessage(err.Error())
	}

	if len(alerts) == 0 {
		return bot.NewTextMessage("No alerts")
	}

	return bot.NewHTMLMessage(c.Formatter.FormatAlerts(alerts, labels))
}

// Silences returns a Markdown formatted NewMessage containing silences with the specified state.
func (c *Client) Silences(state string) string {
	silences, err := c.Alertmanager.Silence.List(context.TODO(), "")
	if err != nil {
		return err.Error()
	}

	md := c.Formatter.FormatSilences(silences, state)

	if md == "" {
		return fmt.Sprintf("No %s silences", state)
	}

	return md
}

// NewSilence creates a new silence and returns the ID.
func (c *Client) NewSilence(author, durationStr string, matchers string) string {
	duration, err := parseDuration(durationStr)
	if err != nil {
		return err.Error()
	}

	silence := types.Silence{
		Matchers:  make(labels.Matchers, len(matchers)),
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(duration),
		CreatedBy: author,
		Comment:   "Created from Matrix",
	}

	// Check if an ID is given instead of matchers
	if len(matchers) == 1 && !strings.ContainsAny(matchers, `=~`) {
		res := c.addSilenceForFingerprint(&silence, matchers)
		if res != "" {
			return res
		}
	} else {
		silence.Matchers, err = labels.ParseMatchers(matchers)
		if err != nil {
			return fmt.Sprintf("Invalid matchers: %s", err)
		}
	}

	id, err := c.Alertmanager.Silence.Set(context.Background(), silence)
	if err != nil {
		return fmt.Sprintf("Error creating silence: %s", err)
	}

	return fmt.Sprintf("Silence created with ID *%s*", id)
}

func (c *Client) addSilenceForFingerprint(silence *types.Silence, fingerprint string) string {
	alert, err := c.Alertmanager.GetAlert(fingerprint)
	if err != nil {
		return err.Error()
	}

	if alert == nil {
		return fmt.Sprintf("No alert with fingerprint %s", fingerprint)
	}

	silence.Matchers = make(labels.Matchers, 0, len(alert.Labels))
	for name, value := range alert.Labels {
		silence.Matchers = append(silence.Matchers, &labels.Matcher{
			Name:  string(name),
			Value: string(value),
		})
	}

	return ""
}

// DelSilence deletes silences.
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
