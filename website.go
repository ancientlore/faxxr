package main

import (
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	templates = template.Must(template.ParseGlob("media/*.html"))
)

func home(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "home.html", nil)
}

var (
	phoneRE       = regexp.MustCompile(`^\+\d+$`)
	phoneReplacer = strings.NewReplacer(" ", "", "-", "", ".", "")
)

func sendFax(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(64 * 1024 * 1024)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var info faxCoverDetails
	info.created = time.Now()
	info.FromName = r.FormValue("fromName")
	info.FromPhone = phoneReplacer.Replace(r.FormValue("fromPhone"))
	info.FromAddr1 = r.FormValue("fromAddr1")
	info.FromAddr2 = r.FormValue("fromAddr2")
	info.ToName = r.FormValue("toName")
	info.ToPhone = phoneReplacer.Replace(r.FormValue("toPhone"))
	info.Subject = r.FormValue("subject")
	info.Text = r.FormValue("text")
	info.Quality = r.FormValue("quality")
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
	if !twilioClient.isWhitelisted(info.FromPhone) {
		http.Error(w, "From phone number is not whitelisted", http.StatusBadRequest)
		return
	}

	// Save media file
	f, hdr, err := r.FormFile("mediaFile")
	defer f.Close()
	ct := hdr.Header.Get("Content-Type")
	ext, err := mime.ExtensionsByType(ct)
	if err != nil || len(ext) < 1 {
		log.Print("Cannot determine file type: ", ct, " assuming PDF")
		ext = []string{".pdf"}
	}
	fn := filepath.Join("tmp", uuid.New().String()+ext[0])
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

	// attach file as image if it isn't a pdf
	if !strings.HasSuffix(fn, ".pdf") {
		info.ImageFile = fn
	}

	// make cover
	cover, err := faxCover("tmp", &info)
	if err != nil {
		log.Print("fax cover: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		os.Remove(fn)
		return
	}

	var finalPdf string

	if strings.HasSuffix(fn, ".pdf") {
		// merge the cover and the pdf
		finalPdf, err = mergePdfs("tmp", []string{cover, fn})
		if err != nil {
			log.Print("merge pdf: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		finalPdf = cover
		err = os.Remove(fn)
		if err != nil {
			log.Print("removing image: ", err)
		}
	}

	err = twilioClient.sendSMS(info.FromPhone, "Reply with OK to approve faxing "+hdr.Filename, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		os.Remove(finalPdf)
		return
	}

	info.pdfFile = strings.TrimPrefix(finalPdf, "tmp/")

	twilioClient.fax.faxQueue <- &info

	templates.ExecuteTemplate(w, "sent.html", nil)
}
