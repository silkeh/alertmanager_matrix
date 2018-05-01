package main

import (
	alertmanager "github.com/prometheus/alertmanager/client"
	"github.com/prometheus/client_golang/api"
)

type amClient struct {
	alert   alertmanager.AlertAPI
	silence alertmanager.SilenceAPI
	status  alertmanager.StatusAPI
}

var am amClient

func startAlertmanagerClient(url string) error {
	c, err := api.NewClient(api.Config{Address: url})
	if err != nil {
		return err
	}
	am = amClient{
		alert:   alertmanager.NewAlertAPI(c),
		silence: alertmanager.NewSilenceAPI(c),
		status:  alertmanager.NewStatusAPI(c),
	}

	return nil
}
