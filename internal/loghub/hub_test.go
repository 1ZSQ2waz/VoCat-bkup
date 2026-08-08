package loghub

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestHubHistoryFiltersAndBounds(t *testing.T) {
	hub := New(slog.NewTextHandler(io.Discard, nil), 100)
	logger := slog.New(hub)
	logger.Debug("hidden")
	logger.Info("modem ready", "device", "ec20")
	logger.Warn("retry", "device", "ec20")

	history := hub.History(10, slog.LevelInfo, "ec20")
	if len(history) != 2 {
		t.Fatalf("History() length = %d, want 2", len(history))
	}
	if history[0].Message != "modem ready" || history[1].Message != "retry" {
		t.Fatalf("History() = %#v", history)
	}
	history[0].Fields["device"] = "changed"
	again := hub.History(10, slog.LevelInfo, "")
	if again[0].Fields["device"] != "ec20" {
		t.Fatal("History() exposed mutable internal fields")
	}
}

func TestHubSubscription(t *testing.T) {
	hub := New(slog.NewTextHandler(io.Discard, nil), 100)
	entries, cancel := hub.Subscribe(1)
	defer cancel()
	record := slog.NewRecord(time.Now(), slog.LevelError, "failure", 0)
	record.Add("stage", "ims")
	if err := hub.Handle(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	select {
	case entry := <-entries:
		if entry.Level != "error" || entry.Fields["stage"] != "ims" {
			t.Fatalf("entry = %#v", entry)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log entry")
	}
}
