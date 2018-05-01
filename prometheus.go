package main

import (
	alertmanager "github.com/prometheus/alertmanager/client"
)

type Alert struct {
	alertmanager.Alert
	Status string
}

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
