package alertmanager

import (
	alertmanager "github.com/prometheus/alertmanager/client"
	"github.com/prometheus/client_golang/api"
)

// Client represends a multi-functional Alertmanager API client
type Client struct {
	Alert   alertmanager.AlertAPI
	Silence alertmanager.SilenceAPI
	Status  alertmanager.StatusAPI
}

// NewClient creates an Alertmanager API client
func NewClient(url string) (*Client, error) {
	c, err := api.NewClient(api.Config{Address: url})
	if err != nil {
		return nil, err
	}
	client := &Client{
		Alert:   alertmanager.NewAlertAPI(c),
		Silence: alertmanager.NewSilenceAPI(c),
		Status:  alertmanager.NewStatusAPI(c),
	}

	return client, nil
}
