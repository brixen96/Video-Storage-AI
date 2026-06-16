package httpclient

import (
	"context"
	"net"
	"net/http"
	"time"
)

// UserAgent is the default User-Agent header for scraper requests.
// Many sites block requests without a browser-like UA string.
const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0"

// DefaultClient is a shared HTTP client with reasonable timeouts.
// Use this instead of http.Get() to prevent goroutines from hanging forever
// when a remote site is slow or unresponsive.
//
// Timeouts:
//   - Dial:          10 seconds
//   - TLS handshake: 5 seconds
//   - Idle conn:     30 seconds
//   - Request total:  30 seconds
var DefaultClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   5,
	},
}

// NewRequest creates a new HTTP request with the User-Agent header set.
func NewRequest(method, url string, body interface{}) (*http.Request, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	return req, nil
}
