package alertmanager

import (
	"fmt"
	"strings"

	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
)

const (
	summaryAnnotation  = "summary"
	resolvedAnnotation = "resolved"
	alertStatus        = "alert"
	resolvedStatus     = "resolved"
	suppressedStatus   = "suppressed"
	silencedStatus     = "silenced"
	severityLabel      = "severity"
	alertNameLabel     = "alertname"
)

// Message represents a message received from Alertmanager via webhook.
type Message struct {
	*webhook.Message
	Alerts []*Alert `json:"alerts"`
}

// Alert represents an Alert received from Alertmanager via webhook.
// It is extended with the `status` attribute, and various convenient functions for formatting.
type Alert struct {
	*template.Alert
}

// AlertName returns the value of the `alertname` label.
func (a *Alert) AlertName() string {
	if v, ok := a.Labels[alertNameLabel]; ok {
		return v
	}

	return ""
}

// StatusString returns a string representing the status.
// This is either `resolved`, `silenced`, the value of the `severity` label, or `alert`.
func (a *Alert) StatusString() string {
	if a.Status == resolvedStatus {
		return resolvedStatus
	}

	if a.Status == suppressedStatus || a.Status == silencedStatus {
		return silencedStatus
	}

	if sev, ok := a.Labels[severityLabel]; ok {
		return sev
	}

	return alertStatus
}

// Summary returns the `summary` annotation when set,
// the `resolved` annotation for `resolved` messages,
// or an empty string if neither annotation is present.
func (a *Alert) Summary() string {
	if v, ok := a.Annotations[summaryAnnotation]; ok {
		return v
	}

	if v, ok := a.Annotations[resolvedAnnotation]; ok && a.Status == resolvedStatus {
		return v
	}

	return ""
}

// LabelString returns a formatted list of message labels in the form {key="value"}.
func (a *Alert) LabelString() string {
	labels := make([]string, 0, len(a.Labels))

	for n, v := range a.Labels {
		labels = append(labels, fmt.Sprintf(`%s=%q`, n, v))
	}

	return "{" + strings.Join(labels, ",") + "}"
}
