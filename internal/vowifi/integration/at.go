package integration

import (
	"context"
	"errors"
	"strings"

	"vocat/internal/device"
	"vocat/internal/modem"
	"vocat/internal/store"
)

type ATDeviceController interface {
	Get(string) (device.Device, error)
	List() []device.Device
	ExecuteAT(context.Context, string, string) (modem.Response, error)
	ExecuteSensitiveAT(context.Context, string, string) (modem.Response, error)
}

// ATMapper lets runtime IDs remain stable configuration IDs even when Linux
// re-enumerates a physical modem under a discovery-derived ID.
type ATMapper struct {
	Store   *store.Store
	Devices ATDeviceController
}

func (mapper ATMapper) Get(configuredID string) (device.Device, error) {
	physicalID, err := mapper.resolve(context.Background(), configuredID)
	if err != nil {
		return device.Device{}, err
	}
	return mapper.Devices.Get(physicalID)
}

func (mapper ATMapper) ExecuteAT(
	ctx context.Context,
	configuredID string,
	command string,
) (modem.Response, error) {
	physicalID, err := mapper.resolve(ctx, configuredID)
	if err != nil {
		return modem.Response{}, err
	}
	return mapper.Devices.ExecuteAT(ctx, physicalID, command)
}

func (mapper ATMapper) ExecuteSensitiveAT(
	ctx context.Context,
	configuredID string,
	command string,
) (modem.Response, error) {
	physicalID, err := mapper.resolve(ctx, configuredID)
	if err != nil {
		return modem.Response{}, err
	}
	return mapper.Devices.ExecuteSensitiveAT(ctx, physicalID, command)
}

func (mapper ATMapper) resolve(
	ctx context.Context,
	configuredID string,
) (string, error) {
	if mapper.Store == nil || mapper.Devices == nil {
		return "", errors.New("vowifi AT mapper is not configured")
	}
	if entry, err := mapper.Devices.Get(configuredID); err == nil && entry.Discovered {
		return entry.ID, nil
	}
	config, err := mapper.Store.Device(ctx, configuredID)
	if err != nil {
		return "", err
	}
	for _, entry := range mapper.Devices.List() {
		if !entry.Discovered {
			continue
		}
		candidate := entry.Candidate
		switch {
		case config.ATPort != "" &&
			(config.ATPort == candidate.ATPort.Path ||
				config.ATPort == candidate.ATPort.OpenPath()):
			return entry.ID, nil
		case config.USBPath != "" && config.USBPath == candidate.USBPath:
			return entry.ID, nil
		case config.ControlDevice != "" &&
			(config.ControlDevice == candidate.QMIControl ||
				config.ControlDevice == candidate.ATPort.OpenPath()):
			return entry.ID, nil
		case config.ModemIMEI != "" && entry.Snapshot != nil &&
			config.ModemIMEI == strings.TrimSpace(entry.Snapshot.IMEI):
			return entry.ID, nil
		}
	}
	return "", device.ErrNotFound
}
