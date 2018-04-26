package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (client *twilio) sendSMS(to, body, mediaURL string) error {
	turl := twilioURL + client.AccountSID + "/Messages.json"

	msgData := url.Values{}
	msgData.Set("To", to)
	msgData.Set("From", client.sms.From)
	msgData.Set("Body", body)
	if mediaURL != "" {
		msgData.Set("MediaUrl", mediaURL)
	}
	if client.sms.StatusCallbackURL != "" {
		msgData.Set("StatusCallback", client.sms.StatusCallbackURL)
	}
	msgDataReader := strings.NewReader(msgData.Encode())

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, _ := http.NewRequest("POST", turl, msgDataReader)
	req.SetBasicAuth(client.AccountSID, client.AuthToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var data map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		log.Print("Unable to decode JSON response: ", err)
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("Message from %q to %q: Send: HTTP %d: %v %v", client.sms.From, to, resp.StatusCode, data["code"], data["message"])
		return err
	}

	log.Printf("Message from %q to %q: %v", client.sms.From, to, data["status"])

	return nil
}

func logSmsStatus(v url.Values) {
	to := v.Get("To")
	from := v.Get("From")
	messageStatus := v.Get("MessageStatus")
	errorCode, _ := strconv.Atoi(v.Get("ErrorCode"))
	log.Printf("Message from %q to %q: %d %s", from, to, errorCode, messageStatus)
}

func smsStatusCallback(w http.ResponseWriter, r *http.Request) {
	logSmsStatus(r.Form)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}
