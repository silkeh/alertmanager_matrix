package bot

import (
	"fmt"
	"strings"

	"github.com/prometheus/alertmanager/types"

	"github.com/silkeh/alertmanager_matrix/alertmanager"
)

const (
	summaryAnnotation  = "summary"
	resolvedAnnotation = "resolved"
	alertStatus        = "alert"
	resolvedStatus     = "resolved"
	suppressedStatus   = "suppressed"
	silencedStatus     = "suppressed"
	severityLabel      = "severity"
	alertNameLabel     = "alertname"
)

// Formatter represents a message formatter with an icon and color set.
type Formatter struct {
	Colors map[string]string
	Icons  map[string]string
}

// NewFormatter creates a new formatter with the default icon set.
func NewFormatter() *Formatter {
	return &Formatter{
		Colors: map[string]string{
			"alert":       "black",
			"information": "blue",
			"warning":     "orange",
			"critical":    "red",
			"resolved":    "green",
			"silenced":    "gray",
		},
		Icons: map[string]string{
			"alert":       "🔔️",
			"information": "ℹ️",
			"warning":     "⚠️",
			"critical":    "🚨",
			"resolved":    "✅",
			"silenced":    "🔕",
		},
	}
}

// icon returns the icon for a string.
func (f *Formatter) icon(t string) string {
	if e, ok := f.Icons[t]; ok {
		return e
	}

	return "❔"
}

// color returns the color for string.
func (f *Formatter) color(t string) string {
	if c, ok := f.Colors[t]; ok {
		return c
	}

	return "gray"
}

// CreateMessage formats a message using the status, name and summary.
func (f *Formatter) CreateMessage(status, name, summary, id string) (plain, html string) {
	icon := f.icon(status)
	color := f.color(status)

	if id != "" {
		id = fmt.Sprintf(" (%s)", id)
	}

	plain = fmt.Sprintf("%s %s %s: %s%s", icon, strings.ToUpper(status), name, summary, id)
	html = fmt.Sprintf(`<font color="%s">%s <b>%s</b> %s:</font> %s%s`,
		color, icon, strings.ToUpper(status), name, summary, id)

	return
}

// FormatAlerts formats alerts as plain text and HTML.
func (f *Formatter) FormatAlerts(alerts []*alertmanager.Alert, labels bool) (string, string) {
	plain := make([]string, len(alerts))
	html := make([]string, len(alerts))

	for i, a := range alerts {
		status := statusString(a)
		summary := summaryString(a)
		alertName := alertNameString(a)

		// Format main message
		plain[i], html[i] = f.CreateMessage(status, alertName, summary, a.Fingerprint)

		// Add labels
		if labels {
			pls := make([]string, 0, len(a.Labels))
			hls := make([]string, 0, len(a.Labels))

			for n, v := range a.Labels {
				pls = append(pls, fmt.Sprintf(`%s=%q`, n, v))
				hls = append(hls, fmt.Sprintf("%s=%q", n, v))
			}

			plain[i] += ", labels: {" + strings.Join(pls, ",") + "}"
			html[i] += "<br/><b>Labels:</b> <code>{" + strings.Join(hls, ",") + "}</code>"
		}
	}

	plainBody := strings.Join(plain, "\n")
	htmlBody := strings.Join(html, "<br/>")

	return plainBody, htmlBody
}

// FormatSilences formats silences as Markdown.
func (f *Formatter) FormatSilences(silences []*types.Silence, state string) (md string) {
	for _, s := range silences {
		if s.Status.State != types.SilenceState(state) {
			continue
		}

		endStr := "Ends"
		if s.Status.State == "expired" {
			endStr = "Ended"
		}

		md += fmt.Sprintf(
			"**Silence %s**  \n%s at %s  \nMatches:`%s`\n\n",
			s.ID,
			endStr,
			s.EndsAt.Format("2006-01-02 15:04:05 MST"),
			s.Matchers.String(),
		)
	}

	return md
}

func statusString(a *alertmanager.Alert) (status string) {
	status = alertStatus
	if a.Status == resolvedStatus {
		status = resolvedStatus
	} else if a.Status == suppressedStatus {
		status = silencedStatus
	} else if sev, ok := a.Labels[severityLabel]; ok {
		status = string(sev)
	}

	return
}

func summaryString(a *alertmanager.Alert) (summary string) {
	if v, ok := a.Annotations[summaryAnnotation]; ok {
		summary = string(v)
	}

	if v, ok := a.Annotations[resolvedAnnotation]; ok && a.Status == resolvedStatus {
		summary = string(v)
	}

	return
}

func alertNameString(a *alertmanager.Alert) (name string) {
	if v, ok := a.Labels[alertNameLabel]; ok {
		name = string(v)
	}

	return
}
