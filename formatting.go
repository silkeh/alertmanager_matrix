package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/prometheus/alertmanager/types"
)

var alertIcons = map[string]string{
	"alert":       "🔔️",
	"information": "ℹ",
	"warning":     "⚠️",
	"critical":    "🚨",
	"ok":          "✅",
	"silenced":    "🔕",
}

var alertColors = map[string]string{
	"alert":       "black",
	"information": "blue",
	"warning":     "orange",
	"critical":    "red",
	"ok":          "green",
	"silenced":    "gray",
}

func emoji(t string) string {
	if e, ok := alertIcons[t]; ok {
		return e
	}
	log.Printf("Unknown status: %s", t)
	return "❔"
}

func color(t string) string {
	if c, ok := alertColors[t]; ok {
		return c
	}
	log.Printf("Unknown status: %s", t)
	return "gray"
}

func createMessage(status, name, summary string) (plain, html string) {
	emoji := emoji(status)
	color := color(status)

	plain = fmt.Sprintf("%s %s %s: %s", emoji, strings.ToUpper(status), name, summary)
	html = fmt.Sprintf(`<font color="%s">%s <b>%s</b> %s:</font> %s`, color, emoji, strings.ToUpper(status), name, summary)

	return
}

func formatAlerts(alerts []*Alert) (string, string) {
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

		plain[i], html[i] = createMessage(status, alertName, summary)
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
