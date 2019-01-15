package alertmanager

import (
	"context"

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

// GetAlerts retrieves all silenced or non-silenced alerts.
func (am Client) GetAlerts(silenced bool) ([]*Alert, error) {
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
func (am Client) GetAlert(id string) (alert *Alert, err error) {
	alerts, err := am.GetAlerts(true)
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
