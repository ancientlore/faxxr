package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/myesui/uuid"
)

type faxReceiveML struct {
	XMLName xml.Name `xml:"Receive"`

	// URL to consult when the fax has been received or has failed.
	Action string `xml:"action,attr,omitempty"`

	// HTTP method to use when requesting the action URL; POST or GET. Defaults to POST.
	Method string `xml:"method,attr,omitempty"`

	// The media type used to store media in the fax media store. Currently, supported
	// values are: application/pdf (the default) and image/tiff.
	MediaType string `xml:"mediaType,attr,omitempty"`

	// What size to interpret received pages as (defaults to letter, US Letter).
	// Supported values: letter, legal, and a4.
	PageSize string `xml:"pageSize,attr,omitempty"`

	// Whether or not to store received media in the fax media store (defaults to true).
	StoreMedia bool `xml:"storeMedia,attr,omitempty"`
}

type faxRejectML struct {
	XMLName xml.Name `xml:"Reject"`
}

type faxML struct {
	XMLName xml.Name      `xml:"Response"`
	Receive *faxReceiveML `xml:"Receive,omitempty"`
	Reject  *faxRejectML  `xml:"Reject,omitempty"`
}

func faxReceive(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("Unable to parse form: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logFaxStatus(r.PostForm)

	enabled := false
	if s, ok := config.Load("fax"); ok && s == "enable" {
		enabled = true
	}

	w.Header().Set("Content-Type", "application/xml")
	data := &faxML{}
	if enabled {
		data.Receive = &faxReceiveML{
			Action:     twilioClient.fax.IncomingDataURL, // URL to post data to
			Method:     "POST",                           // Post response
			MediaType:  "application/pdf",                // PDF
			PageSize:   "",                               // default
			StoreMedia: false,                            // don't store
		}
	} else {
		data.Reject = &faxRejectML{}
	}

	b, err := xml.Marshal(data)
	w.Write([]byte(xml.Header))
	w.Write(b)
}

func faxReceiveFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("Unable to parse form: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logFaxStatus(r.PostForm)

	to := r.PostForm.Get("To")
	from := r.PostForm.Get("From")
	errorCode, _ := strconv.Atoi(r.PostForm.Get("ErrorCode"))
	errorMessage := r.PostForm.Get("ErrorMessage")

	if errorCode != 0 {
		msg := fmt.Sprintf("Failed to receive fax from %q to %q: %d %v ", from, to, errorCode, errorMessage)
		err = twilioClient.sendSMS(twilioClient.sms.From, msg, "")
		if err != nil {
			log.Print("faxReceiveFile: ", err)
		}
	}

	numPages, _ := strconv.Atoi(r.PostForm.Get("NumPages"))
	faxStatus := r.PostForm.Get("FaxStatus")

	// Save media file
	f, hdr, err := r.FormFile("Media")
	defer f.Close()
	ct := hdr.Header.Get("Content-Type")
	ext, err := mime.ExtensionsByType(ct)
	if err != nil || len(ext) < 1 {
		log.Print("Cannot determine file type: ", ct, " assuming PDF")
		ext = []string{".pdf"}
	}
	fn := filepath.Join("tmp", uuid.NewV4().String()+ext[0])
	destf, err := os.Create(fn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(destf, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		destf.Close()
		os.Remove(fn)
		return
	}
	destf.Close()

	msg := fmt.Sprintf("Received fax %q from %q to %q: %v (%d pages)", hdr.Filename, from, to, faxStatus, numPages)
	err = twilioClient.sendSMS(twilioClient.sms.From, msg, "")
	if err != nil {
		log.Print("faxReceiveFile: ", err)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}
