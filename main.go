package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/facebookgo/flagenv"
)

const (
	twilioAPIURL = "https://api.twilio.com/2010-04-01/Accounts/"
)

var (
	flagSID       = flag.String("twilio_sid", "", "Twilio account SID.")
	flagToken     = flag.String("twilio_token", "", "Twilio authorization token.")
	flagFrom      = flag.String("from", "+15716205673", "Phone number to send from.")
	flagAddr      = flag.String("addr", ":9000", "HTTP address to listen on.")
	flagCallback  = flag.String("callback", "http://served.ancientlore.io:9000", "Base URL where callbacks should go.")
	flagWhitelist = flag.String("whitelist", "", "Comma-separated mobile numbers of allowed users.")

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

	twilioClient = &twilio{
		AccountSID: *flagSID,
		AuthToken:  *flagToken,
		HTTPClient: httpClient,
		sms: smsConfig{
			From: *flagFrom,
		},
		fax: faxConfig{
			From:          *flagFrom,
			faxQueue:      make(chan *faxCoverDetails),
			approvalQueue: make(chan string),
			statusQueue:   make(chan string),
			mediaQueue:    make(chan string),
		},
		whitelist: strings.Split(*flagWhitelist, ","),
	}

	if *flagCallback != "" {
		twilioClient.sms.StatusCallbackURL = *flagCallback + "/smsStatus"
		twilioClient.fax.StatusCallbackURL = *flagCallback + "/faxStatus"
		twilioClient.fax.MediaURL = *flagCallback + "/faxMedia/"
		twilioClient.fax.IncomingDataURL = *flagCallback + "/faxReceiveFile"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go twilioClient.faxLoop(ctx)

	// web site
	http.HandleFunc("/", home)
	http.HandleFunc("/sendFax", sendFax)
	http.HandleFunc("/faxMedia/", faxMedia)
	http.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir("media"))))

	// callbacks
	http.HandleFunc("/smsStatus", smsStatusCallback)
	http.HandleFunc("/smsReceive", smsReceive)
	http.HandleFunc("/faxStatus", faxStatusCallback)
	http.HandleFunc("/faxReceive", faxReceive)
	http.HandleFunc("/faxReceiveFile", faxReceiveFile)

	server := &http.Server{
		Addr:         *flagAddr,
		ReadTimeout:  15 * time.Second, // Time to read the request
		WriteTimeout: 15 * time.Second, // Time to write the response
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, os.Kill)
	go func(ctx context.Context) {
		done := ctx.Done()
		select {
		case <-done:
		case sig := <-stop:
			log.Print("main: Received signal ", sig.String())
			d := time.Second * 5
			if sig == os.Kill {
				d = time.Second * 15
			}
			wait, cancel := context.WithTimeout(ctx, d)
			defer cancel()
			err := server.Shutdown(wait)
			if err != nil {
				log.Print("main: ", err)
			}
		}
	}(ctx)

	log.Print("main: Starting on ", *flagAddr)

	// listen for requests and serve responses.
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	log.Print("main: Shutting down")
}
