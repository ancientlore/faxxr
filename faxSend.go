package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func (client *twilio) sendFax(to, mediaURL, quality string) (string, error) {
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
		return "", err
	}
	defer resp.Body.Close()
	var data map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		log.Print("Unable to decode JSON response: ", err)
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("Fax from %q to %q: Send: HTTP %d: %v %v", client.sms.From, to, resp.StatusCode, data["code"], data["message"])
		return "", err
	}

	log.Printf("Fax from %q to %q: %v", client.sms.From, to, data["status"])
	log.Print(data)
	return fmt.Sprint(data["faxSid"]), nil
}

func logFaxStatus(v url.Values) {
	to := v.Get("To")
	from := v.Get("From")
	messageStatus := v.Get("MessageStatus")
	errorCode, _ := strconv.Atoi(v.Get("ErrorCode"))
	errorMessage := v.Get("ErrorMessage")
	log.Printf("Fax from %q to %q: %d %s %v %v", from, to, errorCode, messageStatus, errorMessage, v.Encode())
}

func faxStatusCallback(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("Unable to parse form: ", err)
	} else {
		logFaxStatus(r.PostForm)
		/*
			to := r.PostForm.Get("To")
			from := r.PostForm.Get("From")
			messageStatus := r.PostForm.Get("MessageStatus")
			errorCode, _ := strconv.Atoi(r.PostForm.Get("ErrorCode"))
			err = twilioClient.sendSMS(from, fmt.Sprintf("Fax to %q: %d %v", to, errorCode, messageStatus), "")
			if err != nil {
				log.Print("Unable to send SMS notification: ", err)
			}
		*/
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}

func (client *twilio) faxLoop(ctx context.Context) {
	done := ctx.Done()
	outgoing := make(map[string]*faxCoverDetails)
	for {
		select {
		case <-done:
			return
		case details := <-client.fax.faxQueue:
			outgoing[details.FromPhone] = details
		case number := <-client.fax.approvalQueue:
			details, ok := outgoing[number]
			if ok {
				sid, err := client.sendFax(details.ToPhone, client.fax.MediaURL+details.pdfFile, details.Quality)
				if err != nil {
					log.Print("faxLoop: ", err)
				}
				details.faxSID = sid
			}
			// TODO: we leave it in the map for now - based on status we need to delete it
		}
	}
}

var reValidFile = regexp.MustCompile(`^tmp/[\-a-zA-Z0-9]+\.pdf$`)

func faxMedia(w http.ResponseWriter, r *http.Request) {
	fn := "tmp/" + strings.TrimPrefix(r.URL.Path, "/faxMedia/")
	if reValidFile.MatchString(fn) {
		b, err := ioutil.ReadFile(fn)
		if err == nil {
			w.Header().Set("Content-type", "application/pdf")
			w.Write(b)
			return
		} else {
			log.Print("faxMedia: ", err)
		}
	}

	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
