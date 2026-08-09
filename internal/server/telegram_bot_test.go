package server

import (
	"testing"
	"time"

	"vocat/internal/modem"
)

func TestParseTelegramCommand(t *testing.T) {
	command, remainder := parseTelegramCommand("  /sms@vocat_bot EC20 +447700900123 hello world  ")
	if command != "sms" || remainder != "EC20 +447700900123 hello world" {
		t.Fatalf("parseTelegramCommand() = %q, %q", command, remainder)
	}
	if command, _ := parseTelegramCommand("ordinary message"); command != "" {
		t.Fatalf("non-command parsed as %q", command)
	}
}

func TestSplitTelegramArgumentsPreservesMessageBody(t *testing.T) {
	parts := splitTelegramArguments("  EC20   +447700900123   code with spaces  ", 3)
	if len(parts) != 3 || parts[0] != "EC20" || parts[1] != "+447700900123" || parts[2] != "code with spaces" {
		t.Fatalf("splitTelegramArguments() = %#v", parts)
	}
}

func TestValidTelegramDialNumber(t *testing.T) {
	for _, value := range []string{"10086", "+447700900123", "12345678901234567890"} {
		if !validTelegramDialNumber(value) {
			t.Errorf("validTelegramDialNumber(%q) = false", value)
		}
	}
	for _, value := range []string{"12", "+", "123;ATH", "12 34", "123456789012345678901"} {
		if validTelegramDialNumber(value) {
			t.Errorf("validTelegramDialNumber(%q) = true", value)
		}
	}
}

func TestTelegramPendingActionIsAuthorizedOneShot(t *testing.T) {
	bot := &telegramBot{pending: make(map[string]telegramPendingAction)}
	action := telegramPendingAction{Kind: "call", ChatID: -1001, AdminID: 42, CreatedAt: time.Now()}
	token, err := bot.putPending(action)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := bot.takePending(token, -1001, 41); ok {
		t.Fatal("different administrator consumed pending action")
	}
	if _, ok := bot.takePending(token, -1001, 42); ok {
		t.Fatal("an unauthorized attempt must invalidate the one-time action")
	}
}

func TestFormatTelegramATIncludesFinalResult(t *testing.T) {
	if got := formatTelegramAT(modem.Response{Final: "OK"}); got != "OK" {
		t.Fatalf("formatTelegramAT(OK) = %q", got)
	}
	if got := formatTelegramAT(modem.Response{Lines: []string{"+CLCC: 1"}, Final: "OK"}); got != "+CLCC: 1\nOK" {
		t.Fatalf("formatTelegramAT(lines) = %q", got)
	}
}
