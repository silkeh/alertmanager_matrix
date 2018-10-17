package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/prometheus/alertmanager/types"
)

// alertIcons represent the icons corresponding to the alert status
var alertIcons = map[string]string{
	"alert":       "🔔️",
	"information": "ℹ️",
	"warning":     "⚠️",
	"critical":    "🚨",
	"ok":          "✅",
	"silenced":    "🔕",
}

// alertColors represent the colors corresponding to the alert status
var alertColors = map[string]string{
	"alert":       "black",
	"information": "blue",
	"warning":     "orange",
	"critical":    "red",
	"ok":          "green",
	"silenced":    "gray",
}

// icon returns the icon for a string
func icon(t string) string {
	if e, ok := alertIcons[t]; ok {
		return e
	}
	log.Printf("Unknown status: %s", t)
	return "❔"
}

// color returns the color for string
func color(t string) string {
	if c, ok := alertColors[t]; ok {
		return c
	}
	log.Printf("Unknown status: %s", t)
	return "gray"
}

// createMessage formats a message using the status, name and summary
func createMessage(status, name, summary, id string) (plain, html string) {
	icon := icon(status)
	color := color(status)

	if id != "" {
		id = fmt.Sprintf(" (%s)", id)
	}

	plain = fmt.Sprintf("%s %s %s: %s%s", icon, strings.ToUpper(status), name, summary, id)
	html = fmt.Sprintf(`<font color="%s">%s <b>%s</b> %s:</font> %s%s`,
		color, icon, strings.ToUpper(status), name, summary, id)

	return
}

// formatAlerts formats alerts as plain text and HTML
func formatAlerts(alerts []*Alert, labels bool) (string, string) {
	plain := make([]string, len(alerts))
	html := make([]string, len(alerts))

	for i, a := range alerts {
		status := "alert"
		if a.Status == "resolved" {
			status = "ok"
		} else if a.Status == "suppressed" {
			status = "silenced"
		} else if sev, ok := a.Labels["severity"]; ok {
			status = string(sev)
		}

		summary := ""
		if v, ok := a.Annotations["summary"]; ok {
			summary = string(v)
		}

		alertName := ""
		if v, ok := a.Labels["alertname"]; ok {
			alertName = string(v)
		}

		// Format main message
		plain[i], html[i] = createMessage(status, alertName, summary, a.Fingerprint)

		// Add labels
		if labels {
			pls := make([]string, 0, len(a.Labels))
			hls := make([]string, 0, len(a.Labels))
			for n, v := range a.Labels {
				pls = append(pls, fmt.Sprintf(`%s="%s"`, n, v))
				hls = append(hls, fmt.Sprintf("<code>%s=\"%s\"</code>", n, v))
			}
			plain[i] += ", labels:" + strings.Join(pls, " ")
			html[i] += "<br/><b>Labels:</b> " + strings.Join(hls, " ")
		}
	}

	plainBody := strings.Join(plain, "\n")
	htmlBody := strings.Join(html, "<br/>")

	return plainBody, htmlBody
}

// formatSilences formats silences as Markdown.
func formatSilences(silences []*types.Silence, state string) string {
	md := ""

	for _, s := range silences {
		if s.Status.State != types.SilenceState(state) {
			continue
		}

		endStr := "Ends"
		if s.Status.State == "expired" {
			endStr = "Ended"
		}

		md += fmt.Sprintf(
			"**Silence %s**  \n%s at %s\n\n",
			s.ID,
			endStr,
			s.EndsAt.Format("2006-01-02 15:04"),
		)

		matchers := make([]string, len(s.Matchers))
		for i, m := range s.Matchers {
			sep := "="
			if m.IsRegex {
				sep = "=~"
			}
			matchers[i] = fmt.Sprintf("`%s%s%q`", m.Name, sep, m.Value)
		}
		md += strings.Join(matchers, ", ") + "\n\n"
	}

	return md
}
