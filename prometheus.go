package main

import "time"

type Alert struct {
	Status       string
	GeneratorURL string
	Labels       map[string]string
	Annotations  map[string]string
	StartsAt     time.Time
	EndsAt       time.Time
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
	Alerts            []Alert
}
