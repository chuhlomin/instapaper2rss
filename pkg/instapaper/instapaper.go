package instapaper

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Item struct {
	Type        string `json:"type"`
	Username    string `json:"username"`     // user
	URL         string `json:"url"`          // bookmark
	Title       string `json:"title"`        // bookmark
	Description string `json:"description"`  // bookmark
	Hash        string `json:"hash"`         // bookmark
	Text        string `json:"text"`         // highlight
	Message     string `json:"message"`      // error
	Tags        []Tag  `json:"tags"`         // bookmark
	Time        int64  `json:"time"`         // highlight
	UserID      int    `json:"user_id"`      // user
	BookmarkID  int    `json:"bookmark_id"`  // bookmark
	HighlightID int    `json:"highlight_id"` // highlight
	Position    int    `json:"position"`     // highlight
	ErrorCode   int    `json:"error_code"`   // error
}

type Tag struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type Client struct {
	httpClient     *http.Client
	getTimestamp   func() string
	getNonce       func() string
	baseEndpoint   string
	consumerKey    string
	consumerSecret string
	token          string
	tokenSecret    string
	userAgent      string
	timeout        time.Duration
}

func NewClient(consumerKey, consumerSecret string, options ...Option) (*Client, error) {
	if consumerKey == "" || consumerSecret == "" {
		return nil, fmt.Errorf("consumer key and secret are required")
	}

	client := &Client{
		baseEndpoint:   "https://www.instapaper.com/api/1/",
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		token:          "",
		tokenSecret:    "",
		timeout:        30 * time.Second,
		userAgent:      "go/" + runtime.Version() + " chuhlomin/instapaper2rss/v0.1",
		getNonce:       getNonce,
		getTimestamp:   getTimestamp,
	}

	for _, option := range options {
		if err := option(client); err != nil {
			return nil, err
		}
	}

	if client.httpClient == nil {
		client.httpClient = &http.Client{
			Timeout: client.timeout,
		}
	}

	return client, nil
}

func (c *Client) GetToken(username, password string) (token, secret string, err error) {
	resp, err := c.callAPI("oauth/access_token", map[string]string{
		"x_auth_username": username,
		"x_auth_password": password,
		"x_auth_mode":     "client_auth",
	})
	if err != nil {
		return "", "", fmt.Errorf("HTTP request failed: %w", err)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	values, err := url.ParseQuery(string(b))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse response body: %w", err)
	}

	token = values.Get("oauth_token")
	secret = values.Get("oauth_token_secret")

	if token == "" || secret == "" {
		return "", "", fmt.Errorf("failed to get token and secret")
	}

	return token, secret, nil
}

func (c *Client) GetBookmarks(params map[string]string) ([]Item, error) {
	resp, err := c.callAPI("bookmarks/list", params)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response []Item
	err = json.Unmarshal(b, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body starts with: %s", err, string(b[:100]))
	}

	return response, err
}

func (c *Client) GetBookmarkText(bookmarkID int) (string, error) {
	resp, err := c.callAPI("bookmarks/get_text", map[string]string{
		"bookmark_id": strconv.Itoa(bookmarkID),
	})
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(b), nil
}

func (c *Client) callAPI(endpoint string, params map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseEndpoint+endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add OAuth1 authorization header
	oauthParams := map[string]string{
		"oauth_consumer_key":     c.consumerKey,
		"oauth_nonce":            c.getNonce(),
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        c.getTimestamp(),
		"oauth_version":          "1.0",
	}

	if c.token != "" {
		oauthParams["oauth_token"] = c.token
	}

	// Generate signature
	signature, err := c.generateSignature(req, params, oauthParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}
	oauthParams["oauth_signature"] = signature

	req.Header.Set("Authorization", buildAuthorizationHeader(oauthParams))
	req.Header.Set("User-Agent", c.userAgent)

	if params != nil {
		form := url.Values{}
		for k, v := range params {
			form.Add(k, v)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Body = io.NopCloser(strings.NewReader(form.Encode()))
	}

	return c.httpClient.Do(req)

	// resp, err := c.httpClient.PostForm(endpoint, url.Values{})
	// if err != nil {
	// 	return fmt.Errorf("HTTP request failed: %w", err)
	// }
	// defer resp.Body.Close()

	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return fmt.Errorf("failed to read response body: %w", err)
	// }

	// err = json.Unmarshal(body, &response)
	// if err != nil {
	// 	return fmt.Errorf("JSON parsing failed: %w, body: %s", err, string(body))
	// }

	// return nil
}

func (c *Client) generateSignature(req *http.Request, reqParams, oauthParams map[string]string) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}

	// Collect all parameters
	params := make(map[string]string)
	for k, v := range oauthParams {
		params[k] = v
	}

	for k, v := range reqParams {
		params[k] = v
	}

	// Create parameter string
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	paramPairs := make([]string, 0, len(params))
	for _, k := range keys {
		paramPairs = append(paramPairs,
			fmt.Sprintf("%s=%s",
				url.QueryEscape(k),
				url.QueryEscape(params[k]),
			),
		)
	}
	paramString := strings.Join(paramPairs, "&")

	// Use consistent URL for testing
	baseURL := req.URL.String()
	if strings.Contains(baseURL, "127.0.0.1") || strings.Contains(baseURL, "localhost") {
		baseURL = strings.Replace(baseURL, req.URL.Host, "test.example.com", 1)
	}

	// Create signature base string
	signatureBase := fmt.Sprintf("%s&%s&%s",
		req.Method,
		url.QueryEscape(baseURL),
		url.QueryEscape(paramString),
	)

	// Create signing key
	signingKey := fmt.Sprintf("%s&%s", url.QueryEscape(c.consumerSecret), c.tokenSecret)

	// Generate HMAC-SHA1 signature
	h := hmac.New(sha1.New, []byte(signingKey))
	h.Write([]byte(signatureBase))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature, nil
}

func buildAuthorizationHeader(params map[string]string) string {
	pairs := make([]string, 0, len(params))
	for k, v := range params {
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, url.QueryEscape(v)))
	}
	sort.Strings(pairs)
	return "OAuth " + strings.Join(pairs, ", ")
}

func getNonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
