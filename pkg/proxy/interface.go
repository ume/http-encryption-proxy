package proxy

import (
	"net/http"
	"net/url"
)

// Input is a map of options for a proxy instance
type Input struct {
	Targets     []*Target
	RoutePrefix string
}

// Target is a location where requests will get routed
// based on path prefix and request body will get encrypted
type Target struct {
	Match        func(http.Request) bool
	PathPrefix   string
	PathPrefixes []string
	Destination  *url.URL
	EncryptJSON  bool
}
