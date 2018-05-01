package main

import (
	"log"
	"strings"

	matrix "github.com/matrix-org/gomatrix"
)

const (
	helpMessage = "Available commands are:\n\n" +
		"- `help`: shows this message\n" +
		"- `list`: shows active alerts\n" +
		"- `list all`: shows active and silenced alerts\n" +
		"- `silence`: shows active silences\n" +
		"- `silence add <duration> <matchers>`: create a new silence\n" +
		"- `silence del <ids>`: create a new silence\n"
)

type MatrixMessage struct {
	MsgType       string `json:"msgtype"`
	Body          string `json:"body"`
	FormattedBody string `json:"formatted_body"`
	Format        string `json:"format"`
}

var client *matrix.Client

// Start the Matrix client
func startMatrixClient(homeserver, userID, token string) (err error) {
	// Create a new client
	client, err = matrix.NewClient(homeserver, userID, token)
	if err != nil {
		return
	}

	// Create sync/message handler
	syncer := client.Syncer.(*matrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", messageHandler)

	// Start syncing
	go sync()

	return
}

// Sync thread
func sync() {
	for {
		err := client.Sync()
		if err != nil {
			log.Printf("Sync error: %s", err)
		}
	}
}

// Message handler
func messageHandler(e *matrix.Event) {
	var err error

	// Get message text
	text, ok := e.Body()
	if !ok ||
		e.Sender == client.UserID ||
		!strings.HasPrefix(text, "!alert") {
		return
	}

	// Room to send response to
	room := Room{e.RoomID, client}

	// Get command
	cmd := strings.Split(text, " ")

	// Compress command
	str := ""
	for _, c := range cmd {
		if len(c) > 0 {
			str += string(c[0])
		}
		if len(str) == 3 {
			break
		}
	}

	switch str[1:] {
	case "l":
		err = room.sendAlerts(false)
	case "la":
		err = room.sendAlerts(true)
	case "s":
		err = room.sendSilences()
	case "sa":
		err = room.sendNewSilence(e.Sender, cmd[3:])
	case "sd":
		err = room.sendDelSilence(cmd[3:])
	default:
		err = room.sendMarkdownMessage(helpMessage)
	}

	if err != nil {
		log.Print("Error: ", err)
	}
}
