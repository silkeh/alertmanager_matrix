package main

import (
	"fmt"
	"log"
	"strings"
)

var alertIcons = map[string]string{
	"alert":    "🔔️",
	"warning":  "⚠️",
	"critical": "😱",
	"ok":       "😄",
	"up":       "😄",
	"down":     "😱",
}

var alertColors = map[string]string{
	"alert":    "black",
	"warning":  "orange",
	"critical": "red",
	"ok":       "green",
	"up":       "green",
	"down":     "red",
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

func formatAlerts(alerts []Alert) (string, string) {
	plain := make([]string, len(alerts))
	html := make([]string, len(alerts))

	for i, a := range alerts {
		status := "alert"
		if a.Status == "resolved" {
			status = "ok"
		} else if sev, ok := a.Labels["severity"]; ok {
			status = sev
		}

		summary := ""
		if v, ok := a.Annotations["summary"]; ok {
			summary = v
		}

		alertName := ""
		if v, ok := a.Labels["alertname"]; ok {
			alertName = v
		}

		plain[i], html[i] = createMessage(status, alertName, summary)
	}

	plainBody := strings.Join(plain, "\n")
	htmlBody := strings.Join(html, "<br/>")

	return plainBody, htmlBody
}
