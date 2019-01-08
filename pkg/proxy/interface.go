package proxy

import "net/url"

// Target is a location where requests will get routed
// based on path prefix and request body will get encrypted
type Target struct {
	PathPrefix   string
	PathPrefixes []string
	Destination  *url.URL
	EncryptJSON  bool
}
