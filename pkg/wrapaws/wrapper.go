package wrapaws

import (
	"log"
	"net/http"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type Request struct {
	Host     string            `json:"host"`
	Path     string            `json:"path"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Encoding string            `json:"encoding,omitempty"`
	Body     string            `json:"body"`
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Encoded    bool              `json:"isBase64Encoded,omitemtpy"`
	Body       string            `json:"body"`
}

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	headers    http.Header
	body       *bytes.Buffer
}

func (w *ResponseWriter) Header() http.Header {
	return w.headers
}

func (w *ResponseWriter) Write(p []byte) (n int, err error) {
	n, err = w.body.Write(p)
	return
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func serve(handler http.Handler, req *Request) (res Response, err error) {
	var body []byte
	if req.Encoding == "base64" {
		body, err = base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return
		}
	} else {
		body = []byte(req.Body)
	}

	r, err := http.NewRequest(req.Method, req.Path, bytes.NewReader(body))
	if err != nil {
		return
	}

	for k, v := range req.Headers {
		r.Header.Add(k, v)
		switch strings.ToLower(k) {
		case "host":
			// we need to set `Host` in the request
			// because Go likes to ignore the `Host` header
			// see https://github.com/golang/go/issues/7682
			r.Host = v
		case "content-length":
			contentLength, _ := strconv.ParseInt(v, 10, 64)
			r.ContentLength = contentLength
		case "x-forwarded-for":
		case "x-real-ip":
			r.RemoteAddr = v
		}
	}

	var bodyBuf bytes.Buffer
	w := &ResponseWriter{
		nil,
		http.StatusOK,
		make(http.Header),
		&bodyBuf,
	}

	handler.ServeHTTP(w, r)
	defer r.Body.Close()

	headers := make(map[string]string)
	for k, v := range w.headers {
		for _, s := range v {
			headers[k] = s
		}
	}

	h, _ := json.Marshal(headers)
	log.Printf("[RES-HEADERS] %s\n", string(h))

	headers["x-served-by"] = "http_enc_proxy"

	res = Response{
		StatusCode: w.statusCode,
		Headers:    headers,
		Encoded:    false,
		Body:       string(bodyBuf.Bytes()), // base64.StdEncoding.EncodeToString(bodyBuf.Bytes()),
	}

	return
}

// Maps the `APIGatewayProxyRequest` to a `Request` instance and invokes `Serve()`
func handler(event events.APIGatewayProxyRequest, userHandler http.Handler) (res Response, err error) {
	var req Request

	if event.Body != "" {
		req.Body = event.Body

		if event.IsBase64Encoded {
			req.Encoding = "base64"
		}
	}

	var path string

	if event.PathParameters["proxy"] != "" {
		path += event.PathParameters["proxy"]
	} else if event.Path != "" {
		path += event.Path
	}

	req.Path = path
	req.Method = event.HTTPMethod
	req.Headers = event.Headers

	res, err = serve(userHandler, &req)

	return
}

// ForLambda returns a handler function for AWS Lambda
func ForLambda(userHandler http.Handler) func(events.APIGatewayProxyRequest) (res Response, err error) {
	return func(e events.APIGatewayProxyRequest) (Response, error) {
		return handler(e, userHandler)
	}
}
