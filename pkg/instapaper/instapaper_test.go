package instapaper

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetToken(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check HTTP method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check Content-Type header
		contentType := r.Header.Get("Content-Type")
		expectedContentType := "application/x-www-form-urlencoded"
		if contentType != expectedContentType {
			t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
		}

		// Check User-Agent header
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("User-Agent header is missing")
		}

		// Check OAuth1 header
		if got, want := r.Header.Get("Authorization"),
			"OAuth oauth_consumer_key=\"test_key\", oauth_nonce=\"test_nonce\", oauth_signature=\"BeBhqXc2xaIIEIlQ5UKago7z%2B2g%3D\", oauth_signature_method=\"HMAC-SHA1\", oauth_timestamp=\"1234567890\", oauth_version=\"1.0\""; got != want {
			t.Errorf("Expected Authorization header %q, got %q", want, got)
		}

		// Check form values
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		username := r.Form.Get("x_auth_username")
		if username != "testuser" {
			t.Errorf("Expected username 'testuser', got %s", username)
		}

		password := r.Form.Get("x_auth_password")
		if password != "testpass" {
			t.Errorf("Expected password 'testpass', got %s", password)
		}

		mode := r.Form.Get("x_auth_mode")
		if mode != "client_auth" {
			t.Errorf("Expected mode 'client_auth', got %s", mode)
		}

		// Send mock response
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		_, err := io.WriteString(w, "oauth_token=test_token&oauth_token_secret=test_secret")
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create client with test server URL
	client, err := NewClient(
		"test_key",
		"test_secret",
		WithBaseEndpoint(server.URL+"/"),
		WithNonceGenerator(func() string { return "test_nonce" }),
		WithTimestampGenerator(func() string { return "1234567890" }),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.httpClient = server.Client()

	// Test GetToken
	token, secret, err := client.GetToken("testuser", "testpass")
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	// Check response values
	expectedToken := "test_token"
	if token != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, token)
	}

	expectedSecret := "test_secret"
	if secret != expectedSecret {
		t.Errorf("Expected secret %s, got %s", expectedSecret, secret)
	}
}

func TestGetTokenError(t *testing.T) {
	// Create test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		_, _ = io.WriteString(w, "<html><title>401: Unauthorized</title><body>401: Unauthorized</body></html>") //nolint:errcheck // test only
	}))
	defer server.Close()

	// Create client with test server URL
	client, err := NewClient("test_key", "test_secret", WithBaseEndpoint(server.URL+"/"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.httpClient = server.Client()

	// Test GetToken with invalid credentials
	token, secret, err := client.GetToken("invalid", "invalid")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if token != "" || secret != "" {
		t.Errorf("Expected empty token and secret, got token=%s, secret=%s", token, secret)
	}
}
