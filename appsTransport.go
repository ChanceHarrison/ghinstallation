package ghinstallation

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

// AppsTransport provides a http.RoundTripper by wrapping an existing
// http.RoundTripper and provides GitHub Apps authentication as a
// GitHub App.
//
// Client can also be overwritten, and is useful to change to one which
// provides retry logic if you do experience retryable errors.
//
// See https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/
type AppsTransport struct {
	BaseURL       string            // BaseURL is the scheme and host for GitHub API, defaults to https://api.github.com
	Client        Client            // Client to use to refresh tokens, defaults to http.Client with provided transport
	tr            http.RoundTripper // tr is the underlying roundtripper being wrapped
	key           interface{}       // key is the GitHub App's private key (or any value that is appropriate for the specified signing method)
	signingMethod jwt.SigningMethod // signingMethod specifies how the JWT will be signed (and dictates what key is appropriate)
	appID         int64             // appID is the GitHub App's ID
}

// NewAppsTransportKeyFromFile returns a AppsTransport using a private key from file.
func NewAppsTransportKeyFromFile(tr http.RoundTripper, appID int64, privateKeyFile string) (*AppsTransport, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("could not read private key: %s", err)
	}
	return NewAppsTransport(tr, appID, privateKey)
}

// NewAppsTransport returns a AppsTransport using private key. The key is parsed
// and if any errors occur the error is non-nil.
//
// The provided tr http.RoundTripper should be shared between multiple
// installations to ensure reuse of underlying TCP connections.
//
// The returned Transport's RoundTrip method is safe to be used concurrently.
func NewAppsTransport(tr http.RoundTripper, appID int64, privateKey []byte) (*AppsTransport, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not parse private key: %s", err)
	}
	return NewAppsTransportFromPrivateKey(tr, appID, key), nil
}

// NewAppsTransportFromPrivateKey returns an AppsTransport using a crypto/rsa.(*PrivateKey).
func NewAppsTransportFromPrivateKey(tr http.RoundTripper, appID int64, key *rsa.PrivateKey) *AppsTransport {
	return &AppsTransport{
		BaseURL:       apiBaseURL,
		Client:        &http.Client{Transport: tr},
		tr:            tr,
		key:           key,
		signingMethod: jwt.SigningMethodRS256,
		appID:         appID,
	}
}

// NewAppsTransportCustomSigningMethod returns an AppsTransport using the chosen signingMethod and a compatible key.
func NewAppsTransportCustomSigningMethod(tr http.RoundTripper, appID int64, key interface{}, signingMethod jwt.SigningMethod) (*AppsTransport, error) {
	// Verify that the given key is compatible with the given signingMethod
	_, err := jwt.New(signingMethod).SignedString(key)
	if err != nil {
		return nil, fmt.Errorf("could not sign jwt with given key: %s", err)
	}
	return &AppsTransport{
		BaseURL:       apiBaseURL,
		Client:        &http.Client{Transport: tr},
		tr:            tr,
		key:           key,
		signingMethod: signingMethod,
		appID:         appID,
	}, nil
}

// RoundTrip implements http.RoundTripper interface.
func (t *AppsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// GitHub rejects expiry and issue timestamps that are not an integer,
	// while the jwt-go library serializes to fractional timestamps.
	// Truncate them before passing to jwt-go.
	iss := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	exp := iss.Add(2 * time.Minute)
	claims := &jwt.StandardClaims{
		IssuedAt:  iss.Unix(),
		ExpiresAt: exp.Unix(),
		Issuer:    strconv.FormatInt(t.appID, 10),
	}

	signingMethod := t.signingMethod
	// This should not occur since NewAppsTransportFromPrivateKey was updated to explicitly set the signing method to RS256
	if signingMethod == nil {
		return nil, fmt.Errorf("the AppsTransport's signingMethod is unexpectedly nil")
	}

	bearer := jwt.NewWithClaims(signingMethod, claims)

	ss, err := bearer.SignedString(t.key)
	if err != nil {
		return nil, fmt.Errorf("could not sign jwt: %s", err)
	}

	req.Header.Set("Authorization", "Bearer "+ss)
	req.Header.Add("Accept", acceptHeader)

	resp, err := t.tr.RoundTrip(req)
	return resp, err
}
