package bot

import (
	"gitlab.com/slxh/matrix/alertmanager_matrix/pkg/alertmanager"
)

// Message represents the information for a single alert message.
// It is used for formatting.
type Message struct {
	Alerts     []*alertmanager.Alert
	ShowLabels bool
}
