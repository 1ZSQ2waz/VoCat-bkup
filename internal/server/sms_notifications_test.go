package server

import (
	"strings"
	"testing"
	"time"
)

func TestSMSNotificationTextMatchesUserFacingTemplate(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	previousLocation := time.Local
	time.Local = location
	t.Cleanup(func() { time.Local = previousLocation })
	message := smsNotification{
		DeviceID: "device-1", DeviceLabel: "EC20", Number: "+447386",
		Time: time.Date(2026, 8, 8, 17, 25, 35, 0, location), Content: "你好鸭",
	}
	want := "收到新短信\n设备  EC20\n号码  +447386\n时间  2026-08-08 17:25:35\n内容  你好鸭"
	if got := message.Text(); got != want {
		t.Fatalf("smsNotification.Text() = %q, want %q", got, want)
	}
	if strings.HasPrefix(message.DetailText(), "收到新短信") {
		t.Fatalf("DetailText() unexpectedly repeats the title: %q", message.DetailText())
	}
}

func TestRenderSMSWebhookTemplate(t *testing.T) {
	message := smsNotification{
		DeviceID: "device-1", DeviceName: "客厅", DeviceLabel: "EC20",
		Number: "+447386", Time: time.Unix(1_700_000_000, 0), Content: "hello",
	}
	got := renderSMSWebhookTemplate("{{event}}|{{device_id}}|{{device_name}}|{{device_label}}|{{number}}|{{text}}|{{content}}", message)
	want := "sms.received|device-1|客厅|EC20|+447386|hello|hello"
	if got != want {
		t.Fatalf("renderSMSWebhookTemplate() = %q, want %q", got, want)
	}
}

func TestValidateSMSNotificationConfig(t *testing.T) {
	valid := map[string]map[string]any{
		"bark":     {"urls": []any{"https://api.day.app/key"}},
		"email":    {"smtp_host": "smtp.example.com", "from_address": "from@example.com", "to_addresses": []any{"to@example.com"}},
		"pushplus": {"token": "secret"},
		"webhook":  {"urls": []any{"https://example.com/hook"}},
	}
	for channel, config := range valid {
		if err := validateSMSNotificationConfig(channel, config); err != nil {
			t.Errorf("validateSMSNotificationConfig(%q) error = %v", channel, err)
		}
	}
	if err := validateSMSNotificationConfig("pushplus", map[string]any{}); err == nil {
		t.Fatal("missing Pushplus token was accepted")
	}
}
