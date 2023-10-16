package alertmanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/alertmanager/api/v2/models"

	"gitlab.com/slxh/matrix/alertmanager_matrix/internal/util"
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
	*models.GettableAlert
	Status string `json:"status"`
}

// AlertName returns the value of the `alertname` label.
func (a *Alert) AlertName() string {
	if v, ok := a.GettableAlert.Labels[alertNameLabel]; ok {
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

	if sev, ok := a.Alert.Labels[severityLabel]; ok {
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
	labels := make([]string, 0, len(a.Alert.Labels))

	for n, v := range a.Alert.Labels {
		labels = append(labels, fmt.Sprintf(`%s=%q`, n, v))
	}

	return "{" + strings.Join(labels, ",") + "}"
}

// StartsAt returns the time that the alert starts at.
func (a *Alert) StartsAt() time.Time {
	return time.Time(util.ValueOrDefault(a.GettableAlert.StartsAt))
}

// EndsAt returns the time that the alert ends at.
func (a *Alert) EndsAt() time.Time {
	return time.Time(util.ValueOrDefault(a.GettableAlert.EndsAt))
}

// UpdatedAt returns the time that the alert was last updated at.
func (a *Alert) UpdatedAt() time.Time {
	return time.Time(util.ValueOrDefault(a.GettableAlert.UpdatedAt))
}

// Fingerprint returns the alert fingerprint.
func (a *Alert) Fingerprint() string {
	return util.ValueOrDefault(a.GettableAlert.Fingerprint)
}
