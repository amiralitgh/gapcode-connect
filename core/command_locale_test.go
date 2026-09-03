package core

import "testing"

func TestI18nMenuDescriptionsStayEnglishInFarsiMode(t *testing.T) {
	i18n := NewI18n(LangFarsi)

	if got := i18n.TIn(LangEnglish, MsgBuiltinCmdHelp); got != "Show this help" {
		t.Fatalf("English menu description = %q, want %q", got, "Show this help")
	}
	if got := i18n.T(MsgBuiltinCmdHelp); got == "Show this help" {
		t.Fatal("runtime Farsi translation unexpectedly became English")
	}
}
