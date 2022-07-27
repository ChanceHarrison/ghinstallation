# ghinstallation

[![GoDoc](https://godoc.org/github.com/bradleyfalzon/ghinstallation?status.svg)](https://godoc.org/github.com/bradleyfalzon/ghinstallation)

`ghinstallation` provides `Transport`, which implements `http.RoundTripper` to
provide authentication as an installation for GitHub Apps.

This library is designed to provide automatic authentication for
https://github.com/google/go-github or your own HTTP client.

See
https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/

# Installation

Get the package:

```bash
GO111MODULE=on go get -u github.com/bradleyfalzon/ghinstallation/v2
```

# GitHub Example

```go
import "github.com/bradleyfalzon/ghinstallation/v2"

func main() {
    // Shared transport to reuse TCP connections.
    tr := http.DefaultTransport

    // Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
    itr, err := ghinstallation.NewKeyFromFile(tr, 1, 99, "2016-10-19.private-key.pem")
    if err != nil {
        log.Fatal(err)
    }

    // Use installation transport with github.com/google/go-github
    client := github.NewClient(&http.Client{Transport: itr})
}
```

# GitHub Enterprise Example

For clients using GitHub Enterprise, set the base URL as follows:

```go
import "github.com/bradleyfalzon/ghinstallation/v2"

const GitHubEnterpriseURL = "https://github.example.com/api/v3"

func main() {
    // Shared transport to reuse TCP connections.
    tr := http.DefaultTransport

    // Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
    itr, err := ghinstallation.NewKeyFromFile(tr, 1, 99, "2016-10-19.private-key.pem")
    if err != nil {
        log.Fatal(err)
    }
    itr.BaseURL = GitHubEnterpriseURL

    // Use installation transport with github.com/google/go-github
    client := github.NewEnterpriseClient(GitHubEnterpriseURL, GitHubEnterpriseURL, &http.Client{Transport: itr})
}
```

## What is app ID and installation ID

`app ID` is the GitHub App ID. \
You can check as following : \
Settings > Developer > settings > GitHub App > About item

`installation ID` is a part of WebHook request. \
You can get the number to check the request. \
Settings > Developer > settings > GitHub Apps > Advanced > Payload in Request
tab

```
WebHook request
...
  "installation": {
    "id": `installation ID`
  }
```

# Using an HSM instead of a private key

If it is not practical for your use case to handle the private key directly, you may still be able to use this library.

The `NewTransportCustomSigningMethod` and `NewAppsTransportCustomSigningMethod` functions can be used to create new Transports and AppTransports (respectively) where you specify a jwt.SigningMethod and a compatible key.

To illustrate, here is an example using [gcpjwt](https://github.com/someone1/gcp-jwt-go/v2):

```go
// Shared transport to reuse TCP connections.
tr := http.DefaultTransport

// Prepare to use GCP Cloud KMS
config := &gcpjwt.KMSConfig{
    KeyPath: "name=projects/<project-id>/locations/<location>/keyRings/<key-ring-name>/cryptoKeys/<key-name>/cryptoKeyVersions/<key-version>",
}
key := gcpjwt.NewKMSContext(ctx, config)

// app ID = 1, installation ID = 99, "key" context from above, and the jwt.SigningMethod
itr, err := ghinstallation.NewTransportCustomSigningMethod(tr, 1, 99, key, gcpjwt.SigningMethodKMSRS256)
if err != nil {
    log.Fatal(err)
}

// Use installation transport with github.com/google/go-github
client := github.NewClient(&http.Client{Transport: itr})
```

# License

[Apache 2.0](LICENSE)

# Dependencies

-   [github.com/golang-jwt/jwt-go](https://github.com/golang-jwt/jwt-go)
