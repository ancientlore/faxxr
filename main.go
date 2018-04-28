package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/facebookgo/flagenv"
)

const (
	twilioAPIURL = "https://api.twilio.com/2010-04-01/Accounts/"
)

var (
	flagSID      = flag.String("twilio_sid", "", "Twilio account SID.")
	flagToken    = flag.String("twilio_token", "", "Twilio authorization token.")
	flagFrom     = flag.String("from", "+15716205673", "Phone number to send from.")
	flagAddr     = flag.String("addr", ":9000", "HTTP address to listen on.")
	flagCallback = flag.String("callback", "http://served.ancientlore.io:9000", "Base URL where callbacks should go.")
	flagTo       = flag.String("to", "", "Test phone number to send to.")

	twilioClient *twilio

	httpClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     false,
			DisableCompression:    false,
			MaxIdleConns:          18,
			MaxIdleConnsPerHost:   8,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	config sync.Map
)

func main() {
	flag.Parse()
	flagenv.Parse()

	rand.Seed(time.Now().UnixNano())

	config.Store("fax", "disable")
	config.Store("notify", "on")

	twilioClient = &twilio{
		AccountSID: *flagSID,
		AuthToken:  *flagToken,
		HTTPClient: httpClient,
		sms: smsConfig{
			From: *flagFrom,
		},
		fax: faxConfig{
			From: *flagFrom,
		},
	}

	if *flagCallback != "" {
		twilioClient.sms.StatusCallbackURL = *flagCallback + "/smsStatus"
		twilioClient.fax.StatusCallbackURL = *flagCallback + "/faxStatus"
	}

	// web site
	http.HandleFunc("/", home)
	http.HandleFunc("/sendFax", sendFax)
	http.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir("media"))))

	// callbacks
	http.HandleFunc("/smsStatus", smsStatusCallback)
	http.HandleFunc("/smsReceive", smsReceive)
	http.HandleFunc("/faxStatus", faxStatusCallback)

	server := &http.Server{
		Addr:         *flagAddr,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  10 * time.Second, // Time to read the request
		WriteTimeout: 10 * time.Second, // Time to write the response
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, os.Kill)
	go func(ctx context.Context) {
		done := ctx.Done()
		select {
		case <-done:
		case sig := <-stop:
			log.Print("Received signal ", sig.String())
			d := time.Second * 5
			if sig == os.Kill {
				d = time.Second * 15
			}
			wait, cancel := context.WithTimeout(ctx, d)
			defer cancel()
			err := server.Shutdown(wait)
			if err != nil {
				log.Print(err)
			}
		}
	}(context.Background())

	log.Print("Starting on ", *flagAddr)

	go doStuff(twilioClient)

	// listen for requests and serve responses.
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	log.Print("Shutting down")
}

func doStuff(c *twilio) {
	time.Sleep(2 * time.Second)
	/*
		err := c.sendSMS(*flagTo, "This is a test message. これはテストメッセージです。", "")
		if err != nil {
			log.Print(err)
		}

		faxCover(".", &faxCoverDetails{
			FromPhone: *flagFrom,
			FromName:  "Michael Lore",
			FromAddr1: "1435 Towlston Road",
			FromAddr2: "Vienna, VA 22182",
			ToPhone:   *flagTo,
			ToName:    "Ancient Lore",
			Subject:   "Interesting information",
			Text:      "Please look at this fax. This is some interesting stuff. I promise you will want to read it.\n\nSeriously, look at it!",
		})
	*/
}
