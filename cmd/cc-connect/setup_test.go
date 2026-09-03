package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderSetupConfigWiresGapCodeAndSeparatePlatformIDs(t *testing.T) {
	got := renderSetupConfig(setupAnswers{
		ProjectName:       "my-gapcode",
		WorkDir:           "/tmp/project",
		Language:          "fa",
		TelegramToken:     "telegram-token",
		TelegramProxy:     "socks5://127.0.0.1:10808",
		TelegramAllowFrom: "111",
		BaleToken:         "bale-token",
		BaleAllowFrom:     "222",
	})

	for _, want := range []string{
		`name = "my-gapcode"`,
		`type = "codex"`,
		`cmd = "gapcode"`,
		`work_dir = "/tmp/project"`,
		`language = "fa"`,
		`admin_from = "111,222"`,
		`token = "${TELEGRAM_BOT_TOKEN}"`,
		`proxy = "socks5://127.0.0.1:10808"`,
		`allow_from = "111"`,
		`token = "${BALE_BOT_TOKEN}"`,
		`allow_from = "222"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderSetupConfig() missing %q:\n%s", want, got)
		}
	}
	for _, secret := range []string{"telegram-token", "bale-token"} {
		if strings.Contains(got, secret) {
			t.Fatalf("renderSetupConfig() leaked token %q:\n%s", secret, got)
		}
	}
}

func TestRenderSetupConfigKeepsBootstrapIDsInAdminFrom(t *testing.T) {
	got := renderSetupConfig(setupAnswers{
		ProjectName:       "gapcode",
		WorkDir:           t.TempDir(),
		TelegramToken:     "telegram-token",
		TelegramAllowFrom: telegramUserIDHint,
		BaleToken:         "bale-token",
		BaleAllowFrom:     baleUserIDHint,
	})
	want := `admin_from = "REPLACE_WITH_TELEGRAM_USER_ID,REPLACE_WITH_BALE_USER_ID"`
	if !strings.Contains(got, want) {
		t.Fatalf("renderSetupConfig() admin placeholder = missing %q:\n%s", want, got)
	}
}

func TestWriteSetupConfigUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	err := writeSetupConfig(path, []byte("secret = true\n"))
	if err != nil {
		t.Fatalf("writeSetupConfig() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config permissions = %o, want 600", got)
	}
}

func TestWriteSetupSecretsUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	err := writeSetupSecrets(path, setupAnswers{
		TelegramToken: "telegram-token",
		BaleToken:     "bale-token",
	})
	if err != nil {
		t.Fatalf("writeSetupSecrets() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat secrets: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("secrets permissions = %o, want 600", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read secrets: %v", err)
	}
	if !strings.Contains(string(data), "TELEGRAM_BOT_TOKEN='telegram-token'") ||
		!strings.Contains(string(data), "BALE_BOT_TOKEN='bale-token'") {
		t.Fatalf("secrets file missing expected entries: %s", data)
	}
}

func TestRenderSetupScreenUsesPlainTerminalUI(t *testing.T) {
	var out bytes.Buffer
	renderSetupHeader(&out)
	renderSetupStep(&out, 2, 4, "Bot connections")
	renderSetupWaiting(&out, "Telegram")

	got := out.String()
	for _, want := range []string{
		"GapCode Connect Setup",
		"[2/4] Bot connections",
		"Waiting for a message on Telegram...",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("setup screen missing %q:\n%s", want, got)
		}
	}
}

func TestValidateBotTokenUsesGetMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botvalid-token/getMe" {
			t.Fatalf("request path = %q, want /botvalid-token/getMe", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"gapcode_bot"}}`))
	}))
	defer server.Close()

	username, err := validateBotToken(server.Client(), server.URL, "valid-token")
	if err != nil {
		t.Fatalf("validateBotToken() error = %v", err)
	}
	if username != "gapcode_bot" {
		t.Fatalf("username = %q, want gapcode_bot", username)
	}
}

func TestSetupAnswersRejectEmptyConfiguration(t *testing.T) {
	if err := validateSetupAnswers(setupAnswers{}); err == nil {
		t.Fatal("validateSetupAnswers() accepted empty answers")
	}
}

func TestSetupAnswersRejectFileAsWorkDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	err := validateSetupAnswers(setupAnswers{
		ProjectName:   "gapcode",
		WorkDir:       file,
		TelegramToken: "token",
	})
	if err == nil {
		t.Fatal("validateSetupAnswers() accepted a file as work directory")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("validateSetupAnswers() error = %q, want directory error", err)
	}
}
