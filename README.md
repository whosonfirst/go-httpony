# go-httpony

Utility functions for HTTP ponies written in Go.

## Install

```
make build
```

_See note below inre: [dependencies](#dependencies)._

## Usage

### Crypto

```
package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-httpony/crypto"
)

func main() {

	var key = flag.String("key", "jwPsjM9rfZl73Pt0XURf0t9u8h5ZOpNT", "The key to encrypt and decrypt your text")

	flag.Parse()

	for _, text := range flag.Args() {

		c, err := crypto.NewCrypt(*key)

		if err != nil {
			panic(err)
		}

		enc, err := c.Encrypt(text)

		if err != nil {
			panic(err)
		}

		plain, err := c.Decrypt(enc)

		if err != nil {
			panic(err)
		}

		fmt.Println(text, enc, plain)
	}

}
```

### SSO

```

import (
	"github.com/whosonfirst/go-httpony/sso"
	"net/http"
)

sso_config := "/path/to/ini-config-file.cfg"
endpoint := "localhost:8080"
docroot := "www"
tls_enable := false

sso_provider, err := sso.NewSSOProvider(sso_config, endpoint, docroot, tls_enable)

if err != nil {
	panic(err)
	return
}					

// this is a standard http.HandlerFunc so assume chaining etc. here

sso_handler := sso_provider.SSOHandler()

http.ListenAndServe(endpoint, sso_handler)
		
```

#### SSO Config files

For example:

```
[oauth]
client_id=OAUTH2_CLIENT_ID
client_secret=OAUTH2_CLIENT_SECRET
auth_url=https://example.com/oauth2/request/
token_url=https://example.com/oauth2/token/
api_url=https://example.com/api/
scopes=write

[www]
cookie_name=sso
cookie_secret=SSO_COOKIE_SECRET
```

### TLS

```
import (
	"github.com/whosonfirst/go-httpony/tls"	
	"net/http"
)

// Ensures that httpony/certificates exists in your operating
// system's temporary directory and that its permissions are
// 0700. You do _not_ need to use this if you have your own
// root directory for certificates.

root, err := tls.EnsureTLSRoot()

if err != nil {
	panic(err)
}

// These are self-signed certificates so your browser _will_
// complain about them. All the usual caveats apply.

cert, key, err := tls.GenerateTLSCert(*host, root)
	
if err != nil {
	panic(err)
}

http.ListenAndServeTLS("localhost:443", cert, key, nil)
```

The details of setting up application specific HTTP handlers is left as an exercise to the reader.

## Dependencies

### Vendoring

Vendoring has been disabled for the time being because when trying to load this package as a vendored dependency in _another_ package it all goes pear-shaped with errors like this:

```
make deps
# cd /Users/local/mapzen/mapzen-slippy-map/www-server/vendor/src/github.com/whosonfirst/go-httpony; git submodule update --init --recursive
fatal: no submodule mapping found in .gitmodules for path 'vendor/src/golang.org/x/net'
package github.com/whosonfirst/go-httpony: exit status 128
make: *** [deps] Error 1
```

I have no idea and would welcome suggestions...

