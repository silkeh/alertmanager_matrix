package bot

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/alertmanager/types"
	"gitlab.com/silkeh/matrix-bot"

	"github.com/silkeh/alertmanager_matrix/alertmanager"
)

// Client represents an Alertmanager/Matrix client
type Client struct {
	Matrix       *bot.Client
	Alertmanager *alertmanager.Client
}

// NewClient creates and starts a new Alertmanager/Matrix client
func NewClient(homeserver, userID, token, messageType, rooms, alertmanagerURL string) (client *Client, err error) {
	client = new(Client)

	// Create Alertmanager client
	client.Alertmanager, err = alertmanager.NewClient(alertmanagerURL)
	if err != nil {
		return
	}

	// Matrix bot config
	matrixConfig := &bot.ClientConfig{
		MessageType:      messageType,
		CommandPrefixes:  []string{"!alert", "!alertmanager"},
		IgnoreHighlights: false,
	}

	// Create Matrix client
	client.Matrix, err = bot.NewClient(homeserver, userID, token, matrixConfig)
	if err != nil {
		return
	}

	// Create room list
	if rooms != "" {
		matrixConfig.AllowedRooms = strings.Split(rooms, ",")
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
					return bot.NewMarkdownMessage(c.NewSilence(sender, args))
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

// Run the client in a blocking thread
func (c *Client) Run() error {
	err := c.joinRooms(c.Matrix.Config.AllowedRooms)
	if err != nil {
		return err
	}

	return c.Matrix.Run()
}

// joinRooms joins a list of room IDs or aliases
func (c *Client) joinRooms(roomList []string) error {
	for _, r := range roomList {
		err := c.Matrix.NewRoom(r).Join()
		if err != nil {
			return err
		}
	}

	return nil
}

// Alerts returns all or non-silenced alerts
func (c *Client) Alerts(silenced bool, labels bool) *bot.Message {
	alerts, err := c.Alertmanager.GetAlerts(silenced)
	if err != nil {
		return bot.NewTextMessage(err.Error())
	}
	if len(alerts) == 0 {
		return bot.NewTextMessage("No alerts")
	}

	return bot.NewHTMLMessage(FormatAlerts(alerts, labels))
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
