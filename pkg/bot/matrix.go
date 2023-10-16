// Package bot contains a Matrix Alertmanager bot.
package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/pkg/labels"
	"gitlab.com/slxh/matrix/bot"

	"gitlab.com/slxh/matrix/alertmanager_matrix/internal/util"
	"gitlab.com/slxh/matrix/alertmanager_matrix/pkg/alertmanager"
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
		return nil, fmt.Errorf("error creating Alertmanager client: %w", err)
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
		return nil, fmt.Errorf("error creating Matrix client: %w", err)
	}

	// Create room list
	if config.Rooms != "" {
		matrixConfig.AllowedRooms = strings.Split(config.Rooms, ",")
	}

	// Register commands
	client.Matrix.SetCommand("", client.listOnlyCommand())
	client.Matrix.SetCommand("list", client.listCommand())
	client.Matrix.SetCommand("silence", client.silenceCommand())

	return client, nil
}

// mainCommand returns the `alert` bot command.
func (c *Client) listOnlyCommand() *bot.Command {
	return &bot.Command{
		Summary: "Show active alerts.",
		MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
			return c.Alerts(context.Background(), false, false)
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
				return c.Alerts(context.Background(), true, false)
			},
			Subcommands: map[string]*bot.Command{
				"labels": {
					Summary: "Shows label of active and silenced alerts.",
					MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
						return c.Alerts(context.Background(), true, true)
					},
				},
			},
		},
		"labels": {
			Summary: "Show labels of active alerts.",
			MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
				return c.Alerts(context.Background(), false, true)
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
			return bot.NewMarkdownMessage(c.Silences(context.Background(), "active"))
		},
		Subcommands: map[string]*bot.Command{
			"pending": {
				Summary: "Show pending silences.",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					return bot.NewMarkdownMessage(c.Silences(context.Background(), "pending"))
				},
			},
			"expired": {
				Summary: "Shows expired silences.",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					return bot.NewMarkdownMessage(c.Silences(context.Background(), "expired"))
				},
			},
			"add": {
				Summary: "Create a silence.",
				Description: "Create a silence	using a `duration` and `matcher` or `fingerprint`.\n\n" +
					"A matcher matches job labels, for example: \n" +
					"```\nsilence add 1h job=\"test\",target=~\"test.*\"\n```\n" +
					"Alternative, an alert fingerprint can be given to match all labels of that alert, for example:\n" +
					"```\nsilence add 1h 04e45af092081699\n```\n",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					if len(args) <= 1 {
						return bot.NewTextMessage("Insufficient arguments.")
					}

					return bot.NewMarkdownMessage(c.NewSilence(context.Background(),
						sender, args[0], strings.Join(args[1:], " ")))
				},
			},
			"del": {
				Summary: "Delete a silence by ID.",
				MessageHandler: func(sender, cmd string, args ...string) *bot.Message {
					return bot.NewMarkdownMessage(c.DelSilence(context.Background(), args))
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
	for i, r := range roomList {
		id, err := c.Matrix.NewRoom(r).Join()
		if err != nil {
			return fmt.Errorf("cannot join room %q: %w", r, err)
		}

		roomList[i] = id
	}

	return nil
}

// Alerts returns all or non-silenced alerts.
func (c *Client) Alerts(ctx context.Context, silenced bool, labels bool) *bot.Message {
	alerts, err := c.Alertmanager.GetAlerts(ctx, silenced)
	if err != nil {
		return bot.NewTextMessage(err.Error())
	}

	if len(alerts) == 0 {
		return bot.NewTextMessage("No alerts")
	}

	return bot.NewHTMLMessage(c.Formatter.FormatAlerts(alerts, labels))
}

// Silences returns a Markdown formatted NewMessage containing silences with the specified state.
func (c *Client) Silences(ctx context.Context, state string) string {
	silences, err := c.Alertmanager.GetSilences(ctx)
	if err != nil {
		return fmt.Sprintf("Alertmanager error: %s", err)
	}

	md := c.Formatter.FormatSilences(silences, state)

	if md == "" {
		return fmt.Sprintf("No %s silences", state)
	}

	return md
}

// NewSilence creates a new silence and returns the ID.
func (c *Client) NewSilence(ctx context.Context, author, durationStr string, matchers string) string {
	duration, err := parseDuration(durationStr)
	if err != nil {
		return err.Error()
	}

	silence := alertmanager.Silence{
		GettableSilence: &models.GettableSilence{
			Silence: models.Silence{
				Matchers:  make(models.Matchers, 0, len(matchers)),
				StartsAt:  util.PtrTo(strfmt.DateTime(time.Now())),
				EndsAt:    util.PtrTo(strfmt.DateTime(time.Now().Add(duration))),
				CreatedBy: &author,
				Comment:   util.PtrTo("Created from Matrix"),
			},
		},
	}

	// Check if an ID is given instead of matchers
	if !strings.ContainsAny(matchers, `{"=~!}`) {
		res := c.addSilenceForFingerprint(ctx, &silence.Silence, matchers)
		if res != "" {
			return res
		}
	} else {
		ms, parseErr := labels.ParseMatchers(matchers)
		if parseErr != nil {
			return fmt.Sprintf("Invalid matchers: %s", parseErr)
		}

		silence.SetMatchers(ms)
	}

	id, err := c.Alertmanager.CreateSilence(ctx, silence)
	if err != nil {
		return fmt.Sprintf("Error creating silence: %s", err)
	}

	return fmt.Sprintf("Silence created with ID *%s*", id)
}

func (c *Client) addSilenceForFingerprint(ctx context.Context, silence *models.Silence, fingerprint string) string {
	alert, err := c.Alertmanager.GetAlert(ctx, fingerprint)
	if err != nil {
		return err.Error()
	}

	if alert == nil {
		return fmt.Sprintf("No alert with fingerprint %s", fingerprint)
	}

	silence.Matchers = make(models.Matchers, 0, len(alert.Labels))
	for name, value := range alert.Labels {
		silence.Matchers = append(silence.Matchers, &models.Matcher{
			IsEqual: util.PtrTo(true),
			Name:    util.PtrTo(name),
			Value:   util.PtrTo(value),
		})
	}

	return ""
}

// DelSilence deletes silences.
func (c *Client) DelSilence(ctx context.Context, ids []string) string {
	if len(ids) == 0 {
		return "No silence IDs provided"
	}

	var errors []string

	for _, id := range ids {
		err := c.Alertmanager.DeleteSilence(ctx, id)
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
