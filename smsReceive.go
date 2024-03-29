package main

import (
	"encoding/xml"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

type smsMsg struct {
	XMLName xml.Name `xml:"Message"`
	Body    []string `xml:"Body,omitempty"`
	Media   []string `xml:"Media,omitempty"`
}

type smsML struct {
	XMLName  xml.Name `xml:"Response"`
	Message  *smsMsg  `xml:"Message,omitempty"`
	Redirect string   `xml:"Redirect,omitempty"`
}

var msgReplacer = strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "")

func smsReceive(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("smsReceive: Unable to parse form: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logSmsStatus(r.PostForm)

	msg := ""
	switch msgReplacer.Replace(strings.ToLower(r.PostForm.Get("Body"))) {
	case "help", "options":
		msg = `Msg&Data rates may apply. faxxr options are:
help
options
settings
fax enable|disable
approve
media`
	case "settings":
		msg += "faxxr settings:"
		config.Range(func(k, v interface{}) bool {
			msg += "\n" + k.(string) + " = " + v.(string)
			return true
		})
	case "faxenable", "faxon":
		config.Store("fax", "enable")
		msg = "Receiving faxes enabled."
	case "faxdisable", "faxoff":
		config.Store("fax", "disable")
		msg = "Receiving faxes disabled."
	case "ok", "approve":
		msg = ""
		twilioClient.fax.approvalQueue <- r.PostForm.Get("From")
	case "url", "media":
		msg = ""
		twilioClient.fax.mediaQueue <- r.PostForm.Get("From")
	default:
		msgs := []string{
			"Say what?",
			"I don't understand.",
			"That's not something I can do.",
			"Maybe you should try Google.",
			"My vocabulary is limited.",
			"It's all Greek to me.",
			"You're not speaking my language.",
			"I'm sorry Dave; I can't do that.",
			"You're not the boss of me!",
			"Sorry, I didn't hear you.",
			"Perhasp you mistpyed that?",
			"I'm not Siri.",
			"My name isn't Alexa.",
		}
		msg = msgs[rand.Intn(len(msgs))]
		msg += " Try \"help\" or \"options\" to see what I can do."
	}

	if !twilioClient.isWhitelisted(r.PostForm.Get("From")) {
		msg = "Msg&Data rates may apply."
	}

	w.Header().Set("Content-Type", "application/xml")
	data := &smsML{}
	if msg != "" {
		data.Message = &smsMsg{
			Body: []string{msg},
		}
	}

	b, err := xml.Marshal(data)
	if err != nil {
		log.Print("smsReceive: Unable to marshal response: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(xml.Header))
	w.Write(b)
}
