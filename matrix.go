package main

import (
	matrix "github.com/matrix-org/gomatrix"
)

type MatrixMessage struct {
	MsgType       string `json:"msgtype"`
	Body          string `json:"body"`
	FormattedBody string `json:"formatted_body"`
	Format        string `json:"format"`
}

var client *matrix.Client
var roomID string

func sendMessage(plain, html string) error {
	_, err := client.SendMessageEvent(roomID, "m.room.message",
		&MatrixMessage{
			MsgType:       "m.text",
			Format:        "org.matrix.custom.html",
			Body:          plain,
			FormattedBody: html,
		},
	)

	return err
}
