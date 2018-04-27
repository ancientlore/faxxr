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

func (client *twilio) sendFax(to, mediaURL, quality string) error {
	turl := twilioFaxURL

	msgData := url.Values{}
	msgData.Set("To", to)
	msgData.Set("From", client.fax.From)
	msgData.Set("MediaUrl", mediaURL)
	msgData.Set("StoreMedia", strconv.FormatBool(client.fax.StoreMedia))
	if quality != "" {
		msgData.Set("Quality", quality)
	}
	if client.fax.StatusCallbackURL != "" {
		msgData.Set("StatusCallback", client.fax.StatusCallbackURL)
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
		err = fmt.Errorf("Fax from %q to %q: Send: HTTP %d: %v %v", client.sms.From, to, resp.StatusCode, data["code"], data["message"])
		return err
	}

	log.Printf("Fax from %q to %q: %v", client.sms.From, to, data["status"])

	return nil
}

func logFaxStatus(v url.Values) {
	to := v.Get("To")
	from := v.Get("From")
	messageStatus := v.Get("MessageStatus")
	errorCode, _ := strconv.Atoi(v.Get("ErrorCode"))
	log.Printf("Fax from %q to %q: %d %s", from, to, errorCode, messageStatus, v.Encode())
}

func faxStatusCallback(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("Unable to parse form: ", err)
	} else {
		logFaxStatus(r.PostForm)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}
