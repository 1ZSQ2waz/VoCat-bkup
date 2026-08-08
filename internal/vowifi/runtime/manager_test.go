package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"vocat/internal/vowifi"
)

type fakeSIM struct{}

func (fakeSIM) ReadIdentity(context.Context, string) (vowifi.SIMIdentity, error) {
	return vowifi.SIMIdentity{
		ICCID:           "8944100000000000000",
		HomeMCC:         "234",
		HomeMNC:         "15",
		HomeCountryCode: "GB",
	}, nil
}

type fakeAKA struct{}

func (fakeAKA) CheckReady(context.Context, vowifi.SIMIdentity) (vowifi.AKAEvidence, error) {
	return vowifi.AKAEvidence{Ready: true, Application: "USIM"}, nil
}

func (fakeAKA) Authenticate(context.Context, vowifi.SIMIdentity, vowifi.AKAChallenge) (vowifi.AKAResult, error) {
	return vowifi.AKAResult{}, nil
}

type fakeRadio struct{}

func (fakeRadio) Snapshot(context.Context, string) (vowifi.RadioSnapshot, error) {
	return vowifi.RadioSnapshot{CellularDataEnabled: true, OperatingMode: 1}, nil
}
func (fakeRadio) StopCellularData(context.Context, string) error { return nil }
func (fakeRadio) EnterVoWiFiRFOff(context.Context, string) error { return nil }
func (fakeRadio) Restore(context.Context, string, vowifi.RadioSnapshot) error {
	return nil
}

type fakeProxy struct{}

func (fakeProxy) Resolve(context.Context, vowifi.ProxyRequest) (vowifi.ProxyRoute, error) {
	return vowifi.ProxyRoute{Mode: vowifi.ProxyModeDirect}, nil
}

type fakeTunnelProvider struct{}
type fakeTunnelSession struct{}

func (fakeTunnelProvider) Start(context.Context, vowifi.TunnelRequest) (vowifi.TunnelSession, error) {
	return fakeTunnelSession{}, nil
}
func (fakeTunnelSession) Evidence() vowifi.TunnelEvidence {
	return vowifi.TunnelEvidence{
		Established:   true,
		Name:          "xfrm-test",
		ResponderAUTH: vowifi.ResponderAUTHVerified,
	}
}
func (fakeTunnelSession) Close(context.Context) error { return nil }

type fakeIMSProvider struct{}
type fakeIMSSession struct{}

func (fakeIMSProvider) Start(context.Context, vowifi.IMSRequest) (vowifi.IMSSession, error) {
	return fakeIMSSession{}, nil
}
func (fakeIMSSession) Evidence() vowifi.IMSEvidence {
	return vowifi.IMSEvidence{
		Registered:        true,
		RegistrationState: "registered",
		AssociatedMSISDN:  "+447700900123",
	}
}
func (fakeIMSSession) EnableSMS(context.Context) (vowifi.SMSEvidence, error) {
	return vowifi.SMSEvidence{Ready: true}, nil
}
func (fakeIMSSession) Close(context.Context) error { return nil }

type fakePhones struct{}

func (fakePhones) SaveAssociatedNumber(context.Context, vowifi.PhoneRecord) error {
	return nil
}

func testOrchestrator(t *testing.T, id string) *vowifi.Orchestrator {
	t.Helper()
	orchestrator, err := vowifi.New(vowifi.Dependencies{
		SIM:    fakeSIM{},
		AKA:    fakeAKA{},
		Radio:  fakeRadio{},
		Proxy:  fakeProxy{},
		Tunnel: fakeTunnelProvider{},
		IMS:    fakeIMSProvider{},
		Phones: fakePhones{},
	}, vowifi.Options{DeviceID: id})
	if err != nil {
		t.Fatal(err)
	}
	return orchestrator
}

func TestManagerRunsAndPublishesEnable(t *testing.T) {
	var mu sync.Mutex
	var states []vowifi.State
	manager := New(Options{
		OperationTimeout: time.Second,
		OnState: func(_ context.Context, state vowifi.State) error {
			mu.Lock()
			states = append(states, state)
			mu.Unlock()
			return nil
		},
	})
	t.Cleanup(func() {
		_ = manager.Close(context.Background())
	})
	if err := manager.Register(testOrchestrator(t, "ec20")); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.RequestEnabled("ec20", true); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state, err := manager.State("ec20")
		if err != nil {
			t.Fatal(err)
		}
		if state.Phase == vowifi.PhaseSMSReady {
			if state.PhoneNumber != "+447700900123" {
				t.Fatalf("phone number = %q", state.PhoneNumber)
			}
			mu.Lock()
			published := len(states)
			mu.Unlock()
			if published > 0 {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("enable did not finish")
}

func TestManagerRejectsUnknownDevice(t *testing.T) {
	manager := New(Options{})
	t.Cleanup(func() {
		_ = manager.Close(context.Background())
	})
	if _, err := manager.RequestEnabled("missing", true); !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("error = %v", err)
	}
}

func TestManagerCreatesRuntimeOnDemand(t *testing.T) {
	created := 0
	manager := New(Options{
		Factory: func(_ context.Context, deviceID string) (*vowifi.Orchestrator, error) {
			created++
			return testOrchestrator(t, deviceID), nil
		},
	})
	t.Cleanup(func() {
		_ = manager.Close(context.Background())
	})

	state, err := manager.State("hot-added-ec20")
	if err != nil {
		t.Fatal(err)
	}
	if state.DeviceID != "hot-added-ec20" || created != 1 {
		t.Fatalf("state = %#v, created = %d", state, created)
	}
	if _, err := manager.RequestEnabled("hot-added-ec20", true); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.State("hot-added-ec20"); err != nil {
		t.Fatal(err)
	}
	if created != 1 {
		t.Fatalf("factory calls = %d", created)
	}
}

func TestManagerCoalescesReconnectWhileLifecycleOperationIsBusy(t *testing.T) {
	manager := New(Options{OperationTimeout: time.Second})
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	if err := manager.Register(testOrchestrator(t, "ec20")); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.RequestEnabled("ec20", true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		manager.mu.Lock()
		busy := manager.entries["ec20"].busy
		manager.mu.Unlock()
		if !busy {
			break
		}
		time.Sleep(time.Millisecond)
	}
	before := manager.entries["ec20"].orchestrator.State()
	if before.Phase != vowifi.PhaseSMSReady {
		t.Fatalf("initial phase = %s", before.Phase)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	if _, err := manager.startOperation("ec20", false, func(context.Context, *vowifi.Orchestrator) error {
		close(started)
		<-release
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	if _, err := manager.RequestReconnect("ec20"); err != nil {
		t.Fatalf("queued reconnect error = %v", err)
	}
	// Repeated route changes collapse into the same pending reconnect.
	if _, err := manager.RequestReconnect("ec20"); err != nil {
		t.Fatalf("second queued reconnect error = %v", err)
	}
	if _, err := manager.RequestEnabled("ec20", true); !errors.Is(err, ErrOperationInProgress) {
		t.Fatalf("non-reconnect operation error = %v, want ErrOperationInProgress", err)
	}
	manager.mu.Lock()
	pending := manager.entries["ec20"].reconnectPending
	manager.mu.Unlock()
	if !pending {
		t.Fatal("reconnect was not queued")
	}
	close(release)

	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		manager.mu.Lock()
		busy := manager.entries["ec20"].busy
		manager.mu.Unlock()
		state := manager.entries["ec20"].orchestrator.State()
		if !busy && state.Phase == vowifi.PhaseSMSReady && state.Sequence > before.Sequence {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("queued reconnect did not run after the active operation")
}
