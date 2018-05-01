package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	//log.Print(data)
	return fmt.Sprint(data["sid"]), nil
}

func logFaxStatus(v url.Values) {
	to := v.Get("To")
	from := v.Get("From")
	messageStatus := v.Get("MessageStatus")
	errorCode, _ := strconv.Atoi(v.Get("ErrorCode"))
	errorMessage := v.Get("ErrorMessage")
	log.Printf("Fax from %q to %q: %d %s %v", from, to, errorCode, messageStatus, errorMessage)
}

func faxStatusCallback(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("Unable to parse form: ", err)
	} else {
		logFaxStatus(r.PostForm)
		sid := r.PostForm.Get("FaxSid")
		to := r.PostForm.Get("To")
		faxStatus := r.PostForm.Get("FaxStatus")
		pages, _ := strconv.Atoi(r.PostForm.Get("NumPages"))
		errorCode, _ := strconv.Atoi(r.PostForm.Get("ErrorCode"))
		errorMsg := r.PostForm.Get("ErrorMessage")
		msg := fmt.Sprintf("%s|Fax to %q: %v (%d pages)", sid, to, faxStatus, pages)
		if errorCode != 0 || errorMsg != "" {
			msg += fmt.Sprintf(" %d %v", errorCode, errorMsg)
		}
		twilioClient.fax.statusQueue <- msg
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}

func (client *twilio) faxLoop(ctx context.Context) {
	done := ctx.Done()
	outgoing := make(map[string]*faxCoverDetails)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case details := <-client.fax.faxQueue:
			outgoing[details.FromPhone] = details
		case number := <-client.fax.approvalQueue:
			details, ok := outgoing[number]
			msg := "No pending fax."
			if ok {
				msg = "Fax approved."
				if client.isWhitelisted(details.FromPhone) {
					sid, err := client.sendFax(details.ToPhone, client.fax.MediaURL+details.pdfFile, details.Quality)
					if err != nil {
						log.Print("faxLoop: ", err)
						msg = "Sending failed."
					}
					details.faxSID = sid
				}
			}
			err := client.sendSMS(details.FromPhone, msg, "")
			if err != nil {
				log.Print("faxLoop: ", err)
			}
		case sidMsg := <-client.fax.statusQueue:
			if sidMsg != "" {
				for _, details := range outgoing {
					if strings.HasPrefix(sidMsg, details.faxSID) {
						err := client.sendSMS(details.FromPhone, sidMsg[len(details.faxSID)+1:], "")
						if err != nil {
							log.Print("faxLoop: ", err)
						}
						break
					}
				}
			}
		case <-ticker.C:
			for k, details := range outgoing {
				if time.Since(details.created) > 30*time.Minute {
					log.Print("Removing ", details.pdfFile)
					err := os.Remove("tmp/" + details.pdfFile)
					if err != nil {
						log.Print("faxLoop: ", err)
					}
					delete(outgoing, k)
				}
			}
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
		}
		log.Print("faxMedia: ", err)
	}

	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
