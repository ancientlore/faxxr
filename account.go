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

	// Where to load media from
	MediaURL string

	// Where to send incoming fax data
	IncomingDataURL string

	// Whether to store sent media
	StoreMedia bool

	// a fax we want to send
	faxQueue chan *faxCoverDetails

	// send the SMS phone number where the approval came from to approve a fax.
	approvalQueue chan string

	// fax SID used to sms user about updates
	statusQueue chan string

	// send the SMS phone number to get the media URL of the current pdf.
	mediaQueue chan string
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

	// whitlisted numbers
	whitelist []string
}

func (client *twilio) isWhitelisted(number string) bool {
	for _, v := range client.whitelist {
		if v == number {
			return true
		}
	}
	return false
}

func (client *twilio) ownerNumber() string {
	if len(client.whitelist) > 0 {
		return client.whitelist[0]
	}
	return ""
}
