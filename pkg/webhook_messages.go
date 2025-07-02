package barertc

// WebhookRequest is a JSON request wrapper around all webhook messages.
type WebhookRequest struct {
	Action string
	APIKey string

	// Relevant body per request.
	Report WebhookRequestReport `json:",omitempty"`
}

// WebhookRequestReport is the body for 'report' webhook messages.
type WebhookRequestReport struct {
	FromUsername  string
	AboutUsername string
	Channel       string
	Timestamp     string
	Reason        string
	Message       string
	Comment       string
}
