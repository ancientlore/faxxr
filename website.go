package main

import (
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/myesui/uuid"
)

var (
	templates = template.Must(template.ParseGlob("media/*.html"))
)

func home(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "home.html", nil)
}

var phoneRE = regexp.MustCompile(`^\+\d+`)

func sendFax(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(64 * 1024 * 1024)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var info faxCoverDetails
	info.FromName = r.FormValue("fromName")
	info.FromPhone = strings.TrimSpace(r.FormValue("fromPhone"))
	info.FromAddr1 = r.FormValue("fromAddr1")
	info.FromAddr2 = r.FormValue("fromAddr2")
	info.ToName = r.FormValue("toName")
	info.ToPhone = strings.TrimSpace(r.FormValue("toPhone"))
	info.Subject = r.FormValue("subject")
	info.Text = r.FormValue("text")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if info.FromPhone == "" {
		http.Error(w, "From phone number is required", http.StatusBadRequest)
		return
	}
	if info.ToPhone == "" {
		http.Error(w, "To phone number is required", http.StatusBadRequest)
		return
	}
	if !phoneRE.MatchString(info.FromPhone) {
		http.Error(w, "From phone number is not formatted correctly", http.StatusBadRequest)
		return
	}
	if !phoneRE.MatchString(info.ToPhone) {
		http.Error(w, "To phone number is not formatted correctly", http.StatusBadRequest)
		return
	}
	f, hdr, err := r.FormFile("mediaFile")
	defer f.Close()
	fn := filepath.Join("", uuid.NewV4().String()+".pdf")
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

	cover, err := faxCover("", &info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		os.Remove(fn)
		return
	}

	finalPdf, err := mergePdfs(".", []string{cover, fn})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = twilioClient.sendSMS(info.FromPhone, "Reply with OK to approve faxing "+hdr.Filename, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		os.Remove(finalPdf)
		return
	}

	templates.ExecuteTemplate(w, "sent.html", nil)
}
