package main

import (
	alertmanager "github.com/prometheus/alertmanager/client"
)

// Alert represents an Alert received from Alertmanager via webhook
type Alert struct {
	alertmanager.Alert
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
