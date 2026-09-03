package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	defaultTelegramProxy = "socks5://127.0.0.1:10808"
	telegramUserIDHint   = "REPLACE_WITH_TELEGRAM_USER_ID"
	baleUserIDHint       = "REPLACE_WITH_BALE_USER_ID"
)

type setupAnswers struct {
	ProjectName       string
	WorkDir           string
	Language          string
	TelegramToken     string
	TelegramProxy     string
	TelegramAllowFrom string
	BaleToken         string
	BaleAllowFrom     string
}

type botIdentityResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      struct {
		Username string `json:"username"`
	} `json:"result"`
}

func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	configPath := fs.String("config", "", "path to write config (default: ~/.cc-connect/config.toml)")
	force := fs.Bool("force", false, "overwrite an existing config file")
	_ = fs.Parse(args)

	renderSetupHeader(os.Stdout)
	path := resolveConfigPath(*configPath)
	if _, err := os.Stat(path); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "Config already exists at %s. Use --force to replace it.\n", path)
		os.Exit(1)
	}
	secretsPath := filepath.Join(filepath.Dir(path), "secrets.env")

	answers, err := collectSetupAnswers(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup cancelled: %v\n", err)
		os.Exit(1)
	}
	if err := validateSetupAnswers(answers); err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		os.Exit(1)
	}

	if answers.TelegramToken != "" {
		client, err := setupHTTPClient(answers.TelegramProxy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Telegram proxy validation failed: %v\n", err)
			os.Exit(1)
		}
		username, err := validateBotToken(client, "https://api.telegram.org", answers.TelegramToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Telegram token validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "Telegram bot verified: @%s\n", username)
	}
	if answers.BaleToken != "" {
		client, err := setupHTTPClient("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Bale client setup failed: %v\n", err)
			os.Exit(1)
		}
		username, err := validateBotToken(client, "https://tapi.bale.ai", answers.BaleToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Bale token validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "Bale bot verified: @%s\n", username)
	}

	if err := writeSetupSecrets(secretsPath, answers); err != nil {
		fmt.Fprintf(os.Stderr, "Could not write secrets: %v\n", err)
		os.Exit(1)
	}
	if err := writeSetupConfig(path, []byte(renderSetupConfig(answers))); err != nil {
		fmt.Fprintf(os.Stderr, "Could not write config: %v\n", err)
		os.Exit(1)
	}
	renderSetupStep(os.Stdout, 4, 4, "Done")
	fmt.Fprintf(os.Stdout, "\nWrote private config: %s\n", path)
	fmt.Fprintf(os.Stdout, "Wrote private secrets: %s\n", secretsPath)
	fmt.Fprintf(os.Stdout, "Start: set -a; source %s; set +a; gapcode-connect --config %s\n", secretsPath, path)
	fmt.Fprintln(os.Stdout, "\nFirst test on each enabled platform:")
	fmt.Fprintln(os.Stdout, "  /whoami")
	fmt.Fprintln(os.Stdout, "Then replace the matching REPLACE_WITH_*_USER_ID value in config.toml and restart.")
	fmt.Fprintln(os.Stdout, "\nBale BotFather commands:")
	fmt.Fprintln(os.Stdout, "  Paste the contents of docs/bale-commands-en.txt into Bale @botfather.")
}

func collectSetupAnswers(in *os.File, out, errOut *os.File) (setupAnswers, error) {
	reader := bufio.NewReader(in)
	ask := func(label, fallback string) (string, error) {
		if fallback != "" {
			fmt.Fprintf(out, "%s [%s]: ", label, fallback)
		} else {
			fmt.Fprintf(out, "%s: ", label)
		}
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return "", err
			}
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return fallback, nil
		}
		return line, nil
	}
	secret := func(label string) (string, error) {
		fmt.Fprintf(out, "%s: ", label)
		if term.IsTerminal(int(in.Fd())) {
			value, err := term.ReadPassword(int(in.Fd()))
			fmt.Fprintln(out)
			return strings.TrimSpace(string(value)), err
		}
		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	renderSetupStep(out, 1, 4, "Project")
	project, err := ask("Project name", "gapcode")
	if err != nil {
		return setupAnswers{}, err
	}
	workDir, err := ask("GapCode work directory", cwd)
	if err != nil {
		return setupAnswers{}, err
	}
	language, err := ask("Bridge language (en/fa/auto)", "en")
	if err != nil {
		return setupAnswers{}, err
	}
	renderSetupStep(out, 2, 4, "Bot connections")
	fmt.Fprintln(out, "Leave a bot token blank to skip that platform.")
	fmt.Fprintln(out)
	telegramToken, err := secret("Telegram bot token")
	if err != nil {
		return setupAnswers{}, err
	}
	telegramProxy := ""
	telegramAllow := ""
	if telegramToken != "" {
		telegramProxy, err = ask("Telegram proxy", defaultTelegramProxy)
		if err != nil {
			return setupAnswers{}, err
		}
		renderSetupWaiting(out, "Telegram")
		telegramAllow, err = ask("Telegram allowed user ID (or keep the placeholder and send /whoami after startup)", telegramUserIDHint)
		if err != nil {
			return setupAnswers{}, err
		}
	}
	baleToken, err := secret("Bale bot token")
	if err != nil {
		return setupAnswers{}, err
	}
	baleAllow := ""
	if baleToken != "" {
		renderSetupWaiting(out, "Bale")
		baleAllow, err = ask("Bale allowed user ID (or keep the placeholder and send /whoami after startup)", baleUserIDHint)
		if err != nil {
			return setupAnswers{}, err
		}
	}
	_ = errOut
	renderSetupStep(out, 3, 4, "Access")
	return setupAnswers{
		ProjectName:       project,
		WorkDir:           workDir,
		Language:          normalizeSetupLanguage(language),
		TelegramToken:     telegramToken,
		TelegramProxy:     telegramProxy,
		TelegramAllowFrom: telegramAllow,
		BaleToken:         baleToken,
		BaleAllowFrom:     baleAllow,
	}, nil
}

func renderSetupHeader(out io.Writer) {
	fmt.Fprintln(out, "\nGapCode Connect Setup")
	fmt.Fprintln(out, "Connect GapCode to Telegram and Bale.")
	fmt.Fprintln(out)
}

func renderSetupStep(out io.Writer, step, total int, title string) {
	fmt.Fprintf(out, "[%d/%d] %s\n", step, total, title)
}

func renderSetupWaiting(out io.Writer, platform string) {
	fmt.Fprintf(out, "Waiting for a message on %s...\n", platform)
	fmt.Fprintln(out, "Keep the placeholder if you will use /whoami after startup.")
	fmt.Fprintln(out)
}

func normalizeSetupLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "fa", "farsi":
		return "fa"
	case "auto", "":
		return ""
	default:
		return "en"
	}
}

func validateSetupAnswers(a setupAnswers) error {
	if strings.TrimSpace(a.ProjectName) == "" {
		return errors.New("project name is required")
	}
	if strings.TrimSpace(a.WorkDir) == "" {
		return errors.New("GapCode work directory is required")
	}
	info, err := os.Stat(a.WorkDir)
	if err != nil {
		return fmt.Errorf("work directory %q is not accessible: %w", a.WorkDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("work directory %q is not a directory", a.WorkDir)
	}
	if a.TelegramToken == "" && a.BaleToken == "" {
		return errors.New("enter at least one Telegram or Bale bot token")
	}
	return nil
}

func validateBotToken(client *http.Client, apiBaseURL, token string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := strings.TrimRight(apiBaseURL, "/") + "/bot" + token + "/getMe"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("bot API request failed")
	}
	defer resp.Body.Close()

	var result botIdentityResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("invalid API response")
	}
	if !result.OK {
		if result.Description == "" {
			result.Description = "API rejected the token"
		}
		return "", errors.New(result.Description)
	}
	if result.Result.Username == "" {
		return "", errors.New("API returned no bot username")
	}
	return result.Result.Username, nil
}

func renderSetupConfig(a setupAnswers) string {
	adminIDs := make([]string, 0, 2)
	for _, id := range []string{a.TelegramAllowFrom, a.BaleAllowFrom} {
		id = strings.TrimSpace(id)
		if id != "" {
			adminIDs = append(adminIDs, id)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# GapCode Connect setup\n# Keep the companion secrets.env file private.\n\n")
	fmt.Fprintf(&b, "language = %s\n\n", strconv.Quote(normalizeSetupLanguage(a.Language)))
	fmt.Fprintf(&b, "[[projects]]\nname = %s\nadmin_from = %s\n\n", strconv.Quote(a.ProjectName), strconv.Quote(strings.Join(adminIDs, ",")))
	fmt.Fprintf(&b, "[projects.agent]\ntype = \"codex\"\n\n[projects.agent.options]\ncmd = \"gapcode\"\nwork_dir = %s\nmode = \"yolo\"\n\n", strconv.Quote(a.WorkDir))

	if a.TelegramToken != "" {
		fmt.Fprintf(&b, "[[projects.platforms]]\ntype = \"telegram\"\n\n[projects.platforms.options]\ntoken = %s\nproxy = %s\nallow_from = %s\n\n",
			strconv.Quote("${TELEGRAM_BOT_TOKEN}"), strconv.Quote(a.TelegramProxy), strconv.Quote(a.TelegramAllowFrom))
	}
	if a.BaleToken != "" {
		fmt.Fprintf(&b, "[[projects.platforms]]\ntype = \"bale\"\n\n[projects.platforms.options]\ntoken = %s\napi_base_url = \"https://tapi.bale.ai\"\nallow_from = %s\n",
			strconv.Quote("${BALE_BOT_TOKEN}"), strconv.Quote(a.BaleAllowFrom))
	}
	return b.String()
}

func writeSetupSecrets(path string, a setupAnswers) error {
	var b strings.Builder
	b.WriteString("# Generated by gapcode-connect setup. Keep this file private.\n")
	if a.TelegramToken != "" {
		fmt.Fprintf(&b, "TELEGRAM_BOT_TOKEN=%s\n", shellQuote(a.TelegramToken))
	}
	if a.BaleToken != "" {
		fmt.Fprintf(&b, "BALE_BOT_TOKEN=%s\n", shellQuote(a.BaleToken))
	}
	return writePrivateFile(path, []byte(b.String()))
}

func writeSetupConfig(path string, data []byte) error {
	return writePrivateFile(path, data)
}

func writePrivateFile(path string, data []byte) error {
	if path == "" {
		return errors.New("path is empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func setupHTTPClient(proxyAddress string) (*http.Client, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	if strings.TrimSpace(proxyAddress) == "" {
		return client, nil
	}
	proxyURL, err := url.Parse(proxyAddress)
	if err != nil {
		return nil, err
	}
	client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	return client, nil
}
