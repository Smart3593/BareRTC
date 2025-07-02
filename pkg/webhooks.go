package barertc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/log"
)

// The available and supported webhook event names.
const (
	WebhookReport = "report"
)

// WebhookEnabled checks if the named webhook is enabled.
func WebhookEnabled(name string) bool {
	for _, webhook := range config.Current.WebhookURLs {
		if webhook.Name == name && webhook.Enabled {
			return true
		}
	}
	return false
}

// GetWebhook gets a configured webhook.
func GetWebhook(name string) (config.WebhookURL, bool) {
	for _, webhook := range config.Current.WebhookURLs {
		if webhook.Name == name {
			return webhook, true
		}
	}

	return config.WebhookURL{}, false
}

// PostWebhook submits a JSON body to one of the app's configured webhooks.
//
// Returns the bytes of the response body (hopefully, JSON data) and any errors.
func PostWebhook(name string, payload any) ([]byte, error) {
	webhook, ok := GetWebhook(name)
	if !ok {
		return nil, errors.New("PostWebhook(%s): webhook name %s is not configured")
	} else if !webhook.Enabled {
		return nil, errors.New("PostWebhook(%s): webhook is not enabled")
	}

	// JSON request body.
	jsonStr, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Make the API request to BareRTC.
	var url = webhook.URL
	log.Debug("PostWebhook(%s): to %s we send: %s", name, url, jsonStr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Error("PostWebhook(%s): unexpected response from webhook URL %s (code %d): %s", name, url, resp.StatusCode, body)
		return body, errors.New("unexpected error from webhook URL")
	}

	return body, nil
}

// PostWebhookReport posts a report message via webhook to your website.
func PostWebhookReport(report WebhookRequestReport) error {
	_, err := PostWebhook(WebhookReport, WebhookRequest{
		Action: WebhookReport,
		APIKey: config.Current.AdminAPIKey,
		Report: report,
	})
	return err
}
