package proxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/tidwall/gjson"
	mirror "github.com/ume/api/pkg/envelope"
)

// singleJoiningSlash is copied from httputil.singleJoiningSlash method.
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// NewEncryptionProxy returns an HTTP handler capable of proxying
// multiple targets (hosts or paths) with built-in encryption.
func NewEncryptionProxy(input Input) http.Handler {
	targets := input.Targets

	director := func(req *http.Request) {
		var reqPath string

		if input.RoutePrefix != "" {
			reqPath = strings.Replace(req.URL.Path, input.RoutePrefix, "", 1)
		} else {
			reqPath = req.URL.Path
		}

		log.Printf("[%s-REQ] %s\n", req.Method, reqPath)

		target := findTarget(targets, reqPath)

		targetQuery := target.Destination.RawQuery
		req.URL.Scheme = target.Destination.Scheme
		req.URL.Host = target.Destination.Host
		req.URL.Path = singleJoiningSlash(target.Destination.Path, reqPath)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		// Set the host header to the destination host.
		req.Host = req.URL.Host

		if strings.Contains(req.Header.Get("content-type"), "json") && target.EncryptJSON {
			body, err := ioutil.ReadAll(req.Body)
			defer req.Body.Close()

			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

			if err != nil {
				log.Printf("[%s-ERROR] failed to parse json\n", req.Method)
				return
			}

			result := gjson.GetBytes(body, "properties")
			props := result.Map()

			env := mirror.Envelope{
				Data: make(map[string]string, len(props)),
				Operations: []mirror.Operation{
					{
						Encrypt: &mirror.Encrypt{},
						DataLocation: mirror.DataLocation{
							Source: "*",
						},
					},
				},
			}

			for prop := range props {
				env.Data[prop] = props[prop].String()
			}

			if err := env.Execute(); err != nil {
				log.Printf("[%s-ERROR] failed to execute envelope\n", req.Method)
				return
			}

			encryptedMsg := make(map[string]interface{})
			if err := json.Unmarshal(body, &encryptedMsg); err != nil {
				log.Printf("[%s-ERROR] failed to parse envelope JSON\n", req.Method)
				return
			}

			for prop := range env.Data {
				encryptedMsg["properties"].(map[string]interface{})[prop] = env.Data[prop]
			}

			encryptedJSON, err := json.Marshal(encryptedMsg)
			if err != nil {
				log.Printf("[%s-ERROR] failed to marshal envelope JSON\n", req.Method)
				return
			}

			log.Printf("[%s-ENCRYPT] %s\n", req.Method, string(encryptedJSON))
			req.Body = ioutil.NopCloser(bytes.NewBuffer(encryptedJSON))
			req.ContentLength = int64(len(encryptedJSON))
		}

		finalURL := req.URL.Scheme + "://" + req.URL.Host + req.URL.Path

		log.Printf("[%s-PROXY-REQ] %s\n", req.Method, finalURL)
	}

	modifier := func(res *http.Response) error {
		go func() {
			log.Printf("[%s-PROXY-RES] %d %s\n", res.Request.Method, res.StatusCode, res.Request.URL.String())
		}()

		target := findTarget(targets, res.Request.URL.Path)

		if target.DropGzip {
			res.Header.Del("content-encoding")
			res.Header.Del("Content-Encoding")
		}

		return nil
	}

	return &httputil.ReverseProxy{Director: director, ModifyResponse: modifier}
}

func findTarget(targets []*Target, reqPath string) *Target {
	// Figure out which server to redirect to based on the incoming request.
	var target *Target

	for _, t := range targets {
		if len(t.PathPrefixes) > 0 {
			for _, prefix := range t.PathPrefixes {
				if strings.HasPrefix(reqPath, prefix) {
					target = t
				}
			}
		} else if t.PathPrefix != "" {
			if strings.HasPrefix(reqPath, t.PathPrefix) {
				target = t
			}
		} else {
			target = t
		}

		if target != nil {
			break
		}
	}

	return target
}
