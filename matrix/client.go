package matrix

import (
	matrix "github.com/matrix-org/gomatrix"
	"gopkg.in/russross/blackfriday.v2"
)

// Client represents a Matrix Client
// This is a slightly modified version of gomatrix.Client
type Client struct {
	*matrix.Client
	Syncer      *matrix.DefaultSyncer
	messageType string
}

// Room represents a Matrix Room
type Room struct {
	*Client
	ID string
}

// Message represents a formatted Matrix Message
type Message struct {
	MsgType       string `json:"msgtype"`
	Body          string `json:"body"`
	FormattedBody string `json:"formatted_body,omitempty"`
	Format        string `json:"format,omitempty"`
}

// Event represents a Matrix Event, see gomatrix.Event
type Event matrix.Event

// NewClient returns a configured Matrix Client
func NewClient(homeserverURL, userID, accessToken, messageType string) (c *Client, err error) {
	c = &Client{messageType: messageType}
	c.Client, err = matrix.NewClient(homeserverURL, userID, accessToken)
	if err != nil {
		return
	}
	c.Syncer = c.Client.Syncer.(*matrix.DefaultSyncer)
	return
}

// NewRoom returns a room for a client
func (c *Client) NewRoom(roomID string) *Room {
	return &Room{c, roomID}
}

// SendMessageEvent sends a message event to a room
func (r *Room) SendMessageEvent(eventType string, contentJSON interface{}) (*matrix.RespSendEvent, error) {
	return r.Client.SendMessageEvent(r.ID, eventType, contentJSON)
}

// SendText sends a plain text message
func (r *Room) SendText(plain string) (*matrix.RespSendEvent, error) {
	return r.SendMessageEvent("m.room.message",
		&Message{
			MsgType: r.messageType,
			Body:    plain,
		},
	)
}

// SendHTML sends a plain and HTML formatted message
func (r *Room) SendHTML(plain, html string) (*matrix.RespSendEvent, error) {
	return r.SendMessageEvent("m.room.message",
		&Message{
			MsgType:       r.messageType,
			Format:        "org.matrix.custom.html",
			Body:          plain,
			FormattedBody: html,
		},
	)
}

// SendMarkdown sends a Markdown formatted message as plain text and HTML
func (r *Room) SendMarkdown(md string) (*matrix.RespSendEvent, error) {
	return r.SendHTML(md, Markdown(md))
}

// Markdown formats a message as markdown and returns the HTML representation
func Markdown(md string) string {
	return string(blackfriday.Run([]byte(md),
		blackfriday.WithExtensions(blackfriday.CommonExtensions)))
}
