package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"

	"github.com/ume/http-encryption-proxy/pkg/proxy"
)

var port = flag.String("port", "4800", "bind address")
var debug = flag.Bool("debug", false, "debug mode")
var routePrefix = flag.String("prefix", "", "route prefix")

func main() {
	flag.Parse()

	var cdnURL, apiURL *url.URL
	var err error

	if cdnURL, err = url.Parse("https://cdn.segment.com"); err != nil {
		log.Fatal(err)
	}
	if apiURL, err = url.Parse("https://api.segment.io"); err != nil {
		log.Fatal(err)
	}

	handler := proxy.NewEncryptionProxy(proxy.Input{
		RoutePrefix: *routePrefix,
		Targets: []*proxy.Target{
			&proxy.Target{PathPrefixes: []string{"/v1/projects", "/analytics.js/v1"}, Destination: cdnURL},
			&proxy.Target{Destination: apiURL, EncryptJSON: true},
		},
	})

	if *debug {
		log.Printf("serving proxy at port %v\n", *port)
	}

	log.Fatal(http.ListenAndServe(":"+*port, handler))

}
