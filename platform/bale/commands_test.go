package bale

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestBotFatherCommandListsArePasteable(t *testing.T) {
	linePattern := regexp.MustCompile(`^[a-z][a-z0-9_]* - .+$`)
	for _, name := range []string{"bale-commands-en.txt"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "docs", name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			if len(lines) == 0 {
				t.Fatal("command list is empty")
			}
			foundResume := false
			for _, line := range lines {
				if !linePattern.MatchString(line) {
					t.Errorf("line %q is not in BotFather format", line)
				}
				if strings.HasPrefix(line, "resume - ") {
					foundResume = true
				}
			}
			if !foundResume {
				t.Fatal("command list must include /resume")
			}
		})
	}
}
