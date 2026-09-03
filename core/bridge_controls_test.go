package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleCommand_CdSupportsParentAndHomePaths(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}

	agent := &stubWorkDirAgent{workDir: child}
	e := NewEngine("test", agent, []Platform{p}, "", LangEnglish)
	e.SetAdminFrom("admin")
	msg := &Message{SessionKey: "test:user", UserID: "admin", ReplyCtx: "ctx"}

	if handled := e.handleCommand(p, msg, "/cd .."); !handled {
		t.Fatal("/cd .. should be handled")
	}
	if got := agent.workDir; got != root {
		t.Fatalf("workDir after /cd .. = %q, want %q", got, root)
	}

	if handled := e.handleCommand(p, msg, "/cd ~"); !handled {
		t.Fatal("/cd ~ should be handled")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home dir: %v", err)
	}
	if got := agent.workDir; got != home {
		t.Fatalf("workDir after /cd ~ = %q, want %q", got, home)
	}
}

func TestHandleCommand_CdWithoutArgsReportsCurrentDirectory(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	workDir := t.TempDir()
	e := NewEngine("test", &stubWorkDirAgent{workDir: workDir}, []Platform{p}, "", LangEnglish)
	e.SetAdminFrom("admin")

	if handled := e.handleCommand(p, &Message{
		SessionKey: "test:user",
		UserID:     "admin",
		ReplyCtx:   "ctx",
	}, "/cd"); !handled {
		t.Fatal("/cd should be handled")
	}

	replies := p.getSent()
	if len(replies) != 1 {
		t.Fatalf("/cd replies = %v, want one reply", replies)
	}
	if !strings.Contains(replies[0], workDir) {
		t.Fatalf("/cd reply = %q, want current cwd %q", replies[0], workDir)
	}
	if strings.Contains(replies[0], "History") {
		t.Fatalf("/cd reply = %q, should not expose directory history", replies[0])
	}
}

func TestHandleCommand_PwdReportsPersistentCwd(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	workDir := t.TempDir()
	agent := &stubWorkDirAgent{workDir: workDir}
	e := NewEngine("test", agent, []Platform{p}, "", LangEnglish)
	e.SetAdminFrom("admin")

	handled := e.handleCommand(p, &Message{
		SessionKey: "test:user",
		UserID:     "admin",
		ReplyCtx:   "ctx",
	}, "/pwd")
	if !handled {
		t.Fatal("/pwd should be handled")
	}

	if len(p.getSent()) != 1 || !strings.Contains(p.getSent()[0], workDir) {
		t.Fatalf("/pwd replies = %v, want current cwd %q", p.getSent(), workDir)
	}
}

func TestHandleCommand_NonGapCodeShellControlsAreNotAdded(t *testing.T) {
	p := &stubPlatformEngine{n: "plain"}
	workDir := t.TempDir()
	e := NewEngine("test", &stubWorkDirAgent{workDir: workDir}, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "test:user", UserID: "member", ReplyCtx: "ctx"}

	for _, command := range []string{"/ls", "/mkdir denied", "/launch_gapcode"} {
		if handled := e.handleCommand(p, msg, command); handled {
			t.Fatalf("%s should not be a GapCode Connect command", command)
		}
	}
}

func TestMenuCommandsForPlatform_PublishesGapCodeTUICommandsOnly(t *testing.T) {
	e := NewEngine("test", &stubAgent{}, []Platform{&stubPlatformEngine{n: "plain"}}, "", LangEnglish)
	e.AddCommand("deploy", "Deploy app", "ship it", "", "", "config")

	commands, _ := e.menuCommandsForPlatform("plain")
	names := make(map[string]bool, len(commands))
	for _, command := range commands {
		names[command.Command] = true
	}

	for _, name := range gapCodeTUICommandNames {
		if !names[name] {
			t.Errorf("menu missing %q; commands=%v", name, commands)
		}
	}
	if len(names) != len(gapCodeTUICommandNames) {
		t.Fatalf("menu has %d commands, want exactly %d: %v", len(names), len(gapCodeTUICommandNames), names)
	}
	for _, name := range []string{"ls", "mkdir", "launch_gapcode", "dir", "shell", "show", "deploy"} {
		if names[name] {
			t.Errorf("menu unexpectedly published unrelated command %q", name)
		}
	}
}

func TestGapCodeTUICommandMenuIncludesResume(t *testing.T) {
	commands := gapCodeTUICommands()
	for _, command := range commands {
		if command.Command == "resume" {
			return
		}
	}
	t.Fatalf("GapCode TUI menu does not include /resume: %v", commands)
}

type resumePickerAgent struct {
	stubAgent
	workDir string
	current []AgentSessionInfo
	all     []AgentSessionInfo
}

func (a *resumePickerAgent) GetWorkDir() string { return a.workDir }

func (a *resumePickerAgent) ListSessions(context.Context) ([]AgentSessionInfo, error) {
	return append([]AgentSessionInfo(nil), a.current...), nil
}

func (a *resumePickerAgent) ListAllSessions(context.Context) ([]AgentSessionInfo, error) {
	return append([]AgentSessionInfo(nil), a.all...), nil
}

func TestResumeWithoutArgsOffersGapCodeDirectoryScopes(t *testing.T) {
	p := &stubInlineButtonPlatform{stubPlatformEngine: stubPlatformEngine{n: "telegram"}}
	e := NewEngine("test", &resumePickerAgent{
		workDir: "/tmp/project",
	}, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "telegram:user1", ReplyCtx: "ctx"}

	e.cmdResume(p, msg, nil)

	if len(p.buttonRows) != 1 {
		t.Fatalf("resume button rows = %d, want one row", len(p.buttonRows))
	}
	if len(p.buttonRows[0]) != 2 {
		t.Fatalf("resume buttons = %d, want cwd/all buttons", len(p.buttonRows[0]))
	}
	if got := p.buttonRows[0][0].Data; got != "cmd:/resume cwd" {
		t.Fatalf("cwd button data = %q, want cmd:/resume cwd", got)
	}
	if got := p.buttonRows[0][1].Data; got != "cmd:/resume all" {
		t.Fatalf("all button data = %q, want cmd:/resume all", got)
	}
}

func TestResumeScopeOffersSelectableSessions(t *testing.T) {
	p := &stubInlineButtonPlatform{stubPlatformEngine: stubPlatformEngine{n: "telegram"}}
	current := []AgentSessionInfo{{
		ID:           "thread-cwd-1",
		Summary:      "Fix login",
		MessageCount: 4,
	}}
	e := NewEngine("test", &resumePickerAgent{
		workDir: "/tmp/project",
		current: current,
		all:     append([]AgentSessionInfo(nil), current...),
	}, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "telegram:user1", ReplyCtx: "ctx"}

	e.cmdResume(p, msg, []string{"cwd"})

	if len(p.buttonRows) == 0 {
		t.Fatal("resume cwd did not send session buttons")
	}
	found := false
	for _, row := range p.buttonRows {
		for _, button := range row {
			if button.Data == "cmd:/resume pick cwd 1" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("resume buttons = %#v, want cwd session picker callback", p.buttonRows)
	}
	if !strings.Contains(p.buttonContent, "Current directory") {
		t.Fatalf("resume content = %q, want current-directory scope", p.buttonContent)
	}
}

func TestModelSelectionImmediatelyOffersReasoningEffort(t *testing.T) {
	p := &stubInlineButtonPlatform{stubPlatformEngine: stubPlatformEngine{n: "telegram"}}
	agent := &stubModelModeAgent{model: "gpt-4.1-mini"}
	e := NewEngine("test", agent, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "telegram:user1", ReplyCtx: "ctx"}

	e.cmdModel(p, msg, []string{"switch", "1"})

	foundEffort := false
	for _, row := range p.buttonRows {
		for _, button := range row {
			if strings.HasPrefix(button.Data, "cmd:/reasoning ") {
				foundEffort = true
			}
		}
	}
	if !foundEffort {
		t.Fatalf("model selection buttons = %#v, want reasoning effort picker", p.buttonRows)
	}
}

func TestStatusUsesGapCodeRuntimeFields(t *testing.T) {
	p := &stubPlatformEngine{n: "telegram"}
	agent := &namedStubModelModeAgent{
		stubModelModeAgent: stubModelModeAgent{model: "gpt-5.6-luna", reasoningEffort: "high"},
		name:               "codex",
	}
	e := NewEngine("test", agent, []Platform{p}, "", LangEnglish)
	msg := &Message{SessionKey: "telegram:user1", ReplyCtx: "ctx"}

	e.cmdStatus(p, msg)

	reply := strings.Join(p.getSent(), "\n")
	if strings.Contains(reply, "cc-connect Status") {
		t.Fatalf("status = %q, should not use cc-connect status heading", reply)
	}
	for _, want := range []string{"GapCode", "gpt-5.6-luna", "high", "Directory:"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status = %q, want %q", reply, want)
		}
	}
}
