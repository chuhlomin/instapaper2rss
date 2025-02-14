package instapaper

import (
	"net/http"
	"time"
)

type Option func(*Client) error

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		c.timeout = timeout
		return nil
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *Client) error {
		c.userAgent = userAgent
		return nil
	}
}

func WithToken(token, secret string) Option {
	return func(c *Client) error {
		c.token = token
		c.tokenSecret = secret
		return nil
	}
}

func WithBaseEndpoint(endpoint string) Option {
	return func(c *Client) error {
		c.baseEndpoint = endpoint
		return nil
	}
}

func WithNonceGenerator(f func() string) Option {
	return func(c *Client) error {
		c.getNonce = f
		return nil
	}
}

func WithTimestampGenerator(f func() string) Option {
	return func(c *Client) error {
		c.getTimestamp = f
		return nil
	}
}
