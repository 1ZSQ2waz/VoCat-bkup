package integration

import (
	"context"
	"testing"

	"vocat/internal/device"
	"vocat/internal/modem"
	"vocat/internal/store"
)

type fakeATDevices struct {
	entries     []device.Device
	executedID  string
	sensitiveID string
}

func (devices *fakeATDevices) Get(id string) (device.Device, error) {
	for _, entry := range devices.entries {
		if entry.ID == id {
			return entry, nil
		}
	}
	return device.Device{}, device.ErrNotFound
}

func (devices *fakeATDevices) List() []device.Device {
	return append([]device.Device(nil), devices.entries...)
}

func (devices *fakeATDevices) ExecuteAT(
	_ context.Context,
	id string,
	_ string,
) (modem.Response, error) {
	devices.executedID = id
	return modem.Response{Final: "OK"}, nil
}

func (devices *fakeATDevices) ExecuteSensitiveAT(
	_ context.Context,
	id string,
	_ string,
) (modem.Response, error) {
	devices.sensitiveID = id
	return modem.Response{Final: "OK"}, nil
}

func TestATMapperResolvesConfiguredIDByStableATPath(t *testing.T) {
	database := testStore(t)
	if err := database.UpsertDevice(context.Background(), store.Device{
		ID:     "living-room",
		Name:   "EC20",
		ATPort: "/dev/serial/by-id/usb-ec20-if02",
	}); err != nil {
		t.Fatal(err)
	}
	devices := &fakeATDevices{entries: []device.Device{{
		ID:         "usb-1-2",
		Discovered: true,
		Candidate: modem.Candidate{
			ATPort: modem.Port{
				Path:       "/dev/ttyUSB2",
				StablePath: "/dev/serial/by-id/usb-ec20-if02",
			},
		},
	}}}
	mapper := ATMapper{Store: database, Devices: devices}
	if _, err := mapper.ExecuteAT(
		context.Background(),
		"living-room",
		"AT",
	); err != nil {
		t.Fatal(err)
	}
	if devices.executedID != "usb-1-2" {
		t.Fatalf("ExecuteAT physical ID = %q", devices.executedID)
	}
	if _, err := mapper.ExecuteSensitiveAT(
		context.Background(),
		"living-room",
		"AT+CSIM=1",
	); err != nil {
		t.Fatal(err)
	}
	if devices.sensitiveID != "usb-1-2" {
		t.Fatalf("ExecuteSensitiveAT physical ID = %q", devices.sensitiveID)
	}
}
