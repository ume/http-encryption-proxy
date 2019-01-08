<br/>
<p align="center">
  <strong><code>🔒http-encryption-proxy</code></strong>
</p>

<p align="center">
  Simple HTTP reverse proxy <br/>
  to encrypt JSON payloads in transit
</p>
<br/>
<br/>

### Getting Started

#### Installation

```console
go get -u github.com/ume/http-encryption-proxy
```

### Usage

#### CLI

Enter installation directory (`cd $GOPATH/src/github.com/ume/http-encryption-proxy`) and run:

```console
go run main.go -debug
```

#### API

```go
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

func main() {
	flag.Parse()

	var apiURL *url.URL
	var err error

	if apiURL, err = url.Parse("https://api.thirdparty.com"); err != nil {
		log.Fatal(err)
	}

	handler := proxy.NewEncryptionProxy([]*proxy.Target{
		&proxy.Target{Destination: apiURL, EncryptJSON: true},
	})

	if *debug {
		log.Printf("serving proxy at port %v\n", *port)
	}

	log.Fatal(http.ListenAndServe(":"+*port, handler))
}
```
