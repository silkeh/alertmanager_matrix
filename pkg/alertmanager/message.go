package alertmanager

import (
	"fmt"
	"strings"

	alertmanager "github.com/prometheus/alertmanager/client"
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
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []*Alert          `json:"alerts"`
}

// Alert represents an Alert received from Alertmanager via webhook.
// It is extended with the `status` attribute, and various convenient functions for formatting.
type Alert struct {
	*alertmanager.ExtendedAlert
	Status string `json:"status"`
}

// AlertName returns the value of the `alertname` label.
func (a *Alert) AlertName() string {
	if v, ok := a.ExtendedAlert.Labels[alertNameLabel]; ok {
		return string(v)
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

	if sev, ok := a.Alert.Labels[severityLabel]; ok {
		return string(sev)
	}

	return alertStatus
}

// Summary returns the `summary` annotation when set,
// the `resolved` annotation for `resolved` messages,
// or an empty string if neither annotation is present.
func (a *Alert) Summary() string {
	if v, ok := a.Alert.Annotations[summaryAnnotation]; ok {
		return string(v)
	}

	if v, ok := a.Alert.Annotations[resolvedAnnotation]; ok && a.Status == resolvedStatus {
		return string(v)
	}

	return ""
}

// LabelString returns a formatted list of message labels in the form {key="value"}.
func (a *Alert) LabelString() string {
	labels := make([]string, 0, len(a.Alert.Labels))

	for n, v := range a.Alert.Labels {
		labels = append(labels, fmt.Sprintf(`%s=%q`, n, v))
	}

	return "{" + strings.Join(labels, ",") + "}"
}
