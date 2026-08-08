package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"vocat/internal/store"
)

func TestDeviceProxyBindingPersistsAndReconnectsEnabledVoWiFi(t *testing.T) {
	database, err := store.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.UpsertDevice(context.Background(), store.Device{
		ID: "ec20", Name: "EC20", VoWiFiEnabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.UpsertUpstreamProxy(context.Background(), store.UpstreamProxy{
		ID: "route-1", Name: "Route 1", Addr: "127.0.0.1:1080", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	controller := &fakeVoWiFiController{}
	server := &Server{
		store:               database,
		vowifi:              controller,
		logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		maxRequestBodyBytes: 4096,
	}

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/upstream-proxy-device-bindings/ec20",
		bytes.NewBufferString(`{"upstream_proxy_id":"route-1"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.handleDeviceProxyBinding(response, request, "ec20")
	if response.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", response.Code, response.Body.String())
	}
	binding, err := database.DeviceProxyBinding(context.Background(), "ec20")
	if err != nil || binding.UpstreamProxyID != "route-1" {
		t.Fatalf("binding = %+v, %v", binding, err)
	}
	if controller.reconnects != 1 {
		t.Fatalf("reconnects = %d, want 1", controller.reconnects)
	}

	request = httptest.NewRequest(http.MethodDelete, "/api/upstream-proxy-device-bindings/ec20", nil)
	response = httptest.NewRecorder()
	server.handleDeviceProxyBinding(response, request, "ec20")
	if response.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, body = %s", response.Code, response.Body.String())
	}
	if _, err := database.DeviceProxyBinding(context.Background(), "ec20"); err != store.ErrNotFound {
		t.Fatalf("binding after delete error = %v, want ErrNotFound", err)
	}
	if controller.reconnects != 2 {
		t.Fatalf("reconnects = %d, want 2", controller.reconnects)
	}
}

func TestDeviceProxyBindingRejectsRebindToDifferentUpstream(t *testing.T) {
	database, err := store.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.UpsertDevice(context.Background(), store.Device{
		ID: "ec20", Name: "EC20", VoWiFiEnabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	for _, up := range []store.UpstreamProxy{
		{ID: "route-1", Name: "Route 1", Addr: "127.0.0.1:1080", Enabled: true},
		{ID: "route-2", Name: "Route 2", Addr: "127.0.0.1:1081", Enabled: true},
	} {
		if err := database.UpsertUpstreamProxy(context.Background(), up); err != nil {
			t.Fatal(err)
		}
	}
	server := &Server{
		store:               database,
		vowifi:              &fakeVoWiFiController{},
		logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		maxRequestBodyBytes: 4096,
	}

	// First bind to route-1 succeeds.
	put := func(proxyID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodPut,
			"/api/upstream-proxy-device-bindings/ec20",
			bytes.NewBufferString(`{"upstream_proxy_id":"`+proxyID+`"}`),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.handleDeviceProxyBinding(rec, req, "ec20")
		return rec
	}
	if rec := put("route-1"); rec.Code != http.StatusOK {
		t.Fatalf("initial bind status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Rebind to a different upstream must be rejected with 409.
	rec := put("route-2")
	if rec.Code != http.StatusConflict {
		t.Fatalf("rebind status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}
	binding, err := database.DeviceProxyBinding(context.Background(), "ec20")
	if err != nil || binding.UpstreamProxyID != "route-1" {
		t.Fatalf("binding after rejected rebind = %+v, %v (want route-1 unchanged)", binding, err)
	}

	// Re-binding the SAME upstream stays idempotent (no 409).
	if rec := put("route-1"); rec.Code != http.StatusOK {
		t.Fatalf("idempotent rebind status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
}

