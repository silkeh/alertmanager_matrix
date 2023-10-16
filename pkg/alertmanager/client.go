// Package alertmanager contains a simple Prometheus Alertmanager client.
package alertmanager

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-openapi/strfmt"
	alertmanager "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"

	"gitlab.com/slxh/matrix/alertmanager_matrix/internal/util"
)

// Client represents a multi-functional Alertmanager API client.
type Client struct {
	API *alertmanager.AlertmanagerAPI
}

// NewClient creates an Alertmanager API client.
func NewClient(baseURL string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Alertmanager URL: %w", err)
	}

	if u.Path == "" {
		u.Path = alertmanager.DefaultBasePath
	}

	client := &Client{
		API: alertmanager.NewHTTPClientWithConfig(nil, &alertmanager.TransportConfig{
			Host:     u.Host,
			BasePath: u.Path,
			Schemes:  []string{u.Scheme},
		}),
	}

	return client, nil
}

// GetAlerts retrieves all silenced or non-silenced alerts.
func (am *Client) GetAlerts(ctx context.Context, silenced bool) ([]*Alert, error) {
	alertResp, err := am.API.Alert.GetAlerts(&alert.GetAlertsParams{
		Active:      util.PtrTo(true),
		Inhibited:   util.PtrTo(false),
		Silenced:    &silenced,
		Unprocessed: util.PtrTo(true),
		Context:     ctx,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving commands from alertmanager: %w", err)
	}

	alerts := alertResp.GetPayload()

	// Map alerts to compatible type
	as := make([]*Alert, len(alerts))
	for i, a := range alerts {
		as[i] = &Alert{
			GettableAlert: a,
			Status:        util.ValueOrDefault(a.Status.State),
		}
	}

	return as, nil
}

// GetAlert retrieves an alert with a given ID.
func (am *Client) GetAlert(ctx context.Context, id string) (alert *Alert, err error) {
	alerts, err := am.GetAlerts(ctx, true)
	if err != nil {
		return nil, err
	}

	for _, a := range alerts {
		if a.Fingerprint() == id {
			alert = a
			break
		}
	}

	return
}

// GetSilences returns a list of silences from Alertmanager.
func (am *Client) GetSilences(ctx context.Context) ([]*models.GettableSilence, error) {
	silencesResp, err := am.API.Silence.GetSilences(&silence.GetSilencesParams{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("error retrieving silences: %w", err)
	}

	return silencesResp.GetPayload(), nil
}

// CreateSilence creates the given silence.
func (am *Client) CreateSilence(ctx context.Context, s models.Silence) (string, error) {
	resp, err := am.API.Silence.PostSilences(&silence.PostSilencesParams{
		Silence: &models.PostableSilence{Silence: s},
		Context: ctx,
	})
	if err != nil {
		return "", fmt.Errorf("error creating silence: %w", err)
	}

	return resp.GetPayload().SilenceID, nil
}

// DeleteSilence deletes the silence with the given ID.
func (am *Client) DeleteSilence(ctx context.Context, id string) error {
	_, err := am.API.Silence.DeleteSilence(&silence.DeleteSilenceParams{
		SilenceID: strfmt.UUID(id),
		Context:   ctx,
	})
	if err != nil {
		return fmt.Errorf("error deleting silence: %w", err)
	}

	return nil
}
