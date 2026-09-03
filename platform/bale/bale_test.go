package bale

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/amiralitgh/gapcode-connect/core"
	"github.com/go-telegram/bot/models"
)

func TestBalePlatformIsRegistered(t *testing.T) {
	p, err := core.CreatePlatform("bale", map[string]any{
		"token":        "test-token",
		"api_base_url": "http://127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CreatePlatform(bale) error = %v", err)
	}
	if p.Name() != "bale" {
		t.Fatalf("platform name = %q, want bale", p.Name())
	}
}

func TestBalePlatformUsesNativeEndpoint(t *testing.T) {
	p, err := New(map[string]any{"token": "test-token"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	bale, ok := p.(*Platform)
	if !ok {
		t.Fatalf("platform type = %T, want *Platform", p)
	}
	if bale.apiBaseURL != "https://tapi.bale.ai" {
		t.Fatalf("apiBaseURL = %q, want https://tapi.bale.ai", bale.apiBaseURL)
	}
}

func TestBalePlatformAllowsTestEndpointOverride(t *testing.T) {
	p, err := New(map[string]any{
		"token":        "test-token",
		"api_base_url": "http://127.0.0.1:1234/",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	bale := p.(*Platform)
	if bale.apiBaseURL != "http://127.0.0.1:1234" {
		t.Fatalf("apiBaseURL = %q, want trimmed override", bale.apiBaseURL)
	}
}

func TestBalePlatformRejectsRemoteHTTPOverride(t *testing.T) {
	if _, err := New(map[string]any{
		"token":        "test-token",
		"api_base_url": "http://example.com",
	}); err == nil {
		t.Fatal("New() accepted a remote HTTP api_base_url")
	}
}

func TestBaleCommandMenuUnsupported(t *testing.T) {
	if !isBaleCommandMenuUnsupported(fmt.Errorf("error response from telegram for method setMyCommands, 501 Not Implemented (Coming soon...)")) {
		t.Fatal("expected Bale command-menu 501 to be treated as optional")
	}
	if isBaleCommandMenuUnsupported(fmt.Errorf("error response from telegram for method setMyCommands, 400 Bad Request")) {
		t.Fatal("unexpectedly treated a normal API error as optional")
	}
}

func TestBaleIdentityDiscoveryCommand(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "/whoami", want: true},
		{text: "/WHOAMI", want: true},
		{text: "/whoami@psp_gapcode_bot", want: true},
		{text: "/whoami extra", want: false},
		{text: "whoami", want: false},
		{text: "/status", want: false},
	}

	for _, tt := range tests {
		if got := isIdentityDiscoveryCommand(tt.text); got != tt.want {
			t.Errorf("isIdentityDiscoveryCommand(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestDefaultBaleBotUsesConfiguredServerURL(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Bale","username":"test_bale_bot"}}`))
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, me, _, err := defaultNewBotAtURL(
		"test-token",
		func(context.Context, *models.Update) {},
		server.Client(),
		server.URL,
	)
	if err != nil {
		t.Fatalf("defaultNewBotAtURL() error = %v", err)
	}
	if me.Username != "test_bale_bot" {
		t.Fatalf("bot username = %q, want test_bale_bot", me.Username)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 2 || paths[0] != "/bottest-token/getMe" || paths[1] != "/bottest-token/getMe" {
		t.Fatalf("request paths = %v, want both getMe requests on configured server", paths)
	}
}
