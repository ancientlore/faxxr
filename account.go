package main

import "net/http"

const (
	twilioSMSURL = "https://api.twilio.com/2010-04-01/Accounts/"
	twilioFaxURL = "https://fax.twilio.com/v1/Faxes"
)

type smsConfig struct {
	// From phone number
	From string

	// Status callback URL
	StatusCallbackURL string
}

type faxConfig struct {
	// From phone number
	From string

	// Status callback URL
	StatusCallbackURL string

	// Whether to store sent media
	StoreMedia bool
}

// twilio is a Twilio client.
type twilio struct {
	// AccountSID is the Twilio account ID.
	AccountSID string

	// AuthToken is the Twilio authorization token.
	AuthToken string

	// If provided, the Client will use this HTTP client.
	HTTPClient *http.Client

	// SMS settings
	sms smsConfig

	// fax settings
	fax faxConfig
}
