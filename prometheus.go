package main

import (
	"context"
	alertmanager "github.com/prometheus/alertmanager/client"
)

// Alert represents an Alert received from Alertmanager via webhook
type Alert struct {
	*alertmanager.ExtendedAlert
	Status string
}

// Message represents a message received from Alertmanager via webhook
type Message struct {
	Version           string
	GroupKey          string
	Status            string
	Receiver          string
	GroupLabels       map[string]string
	CommonLabels      map[string]string
	CommonAnnotations map[string]string
	ExternalURL       string
	Alerts            []*Alert
}

// GetAlerts retrieves all silenced or non-silenced alerts.
func GetAlerts(silenced bool) ([]*Alert, error) {
	alerts, err := am.Alert.List(context.TODO(), "", "",
		silenced, false, true, true)
	if err != nil {
		return nil, err
	}

	// Map alerts to compatible type
	as := make([]*Alert, len(alerts))
	for i, a := range alerts {
		as[i] = &Alert{
			ExtendedAlert: a,
			Status:        string(a.Status.State),
		}
	}

	return as, nil
}

// GetAlert retrieves an alert with a given ID
func GetAlert(id string) (alert *Alert, err error) {
	alerts, err := GetAlerts(true)
	if err != nil {
		return nil, err
	}

	for _, a := range alerts {
		if a.Fingerprint == id {
			alert = a
			break
		}
	}

	return
}
