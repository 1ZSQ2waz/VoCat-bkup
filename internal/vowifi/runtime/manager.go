// Package runtime owns the long-lived VoWiFi orchestrators used by the
// service. It keeps HTTP requests short while preserving every evidence-backed
// state transition through the supplied state callback.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"vocat/internal/vowifi"
)

var (
	ErrNotRegistered       = errors.New("vowifi runtime: device is not registered")
	ErrOperationInProgress = errors.New("vowifi runtime: an operation is already in progress")
	ErrClosed              = errors.New("vowifi runtime: manager is closed")
)

const defaultOperationTimeout = 2 * time.Minute

type StateHandler func(context.Context, vowifi.State) error
type OrchestratorFactory func(context.Context, string) (*vowifi.Orchestrator, error)

type Options struct {
	Logger           *slog.Logger
	OperationTimeout time.Duration
	OnState          StateHandler
	Factory          OrchestratorFactory
}

type Manager struct {
	ctx              context.Context
	cancel           context.CancelFunc
	logger           *slog.Logger
	operationTimeout time.Duration
	onState          StateHandler
	factory          OrchestratorFactory

	mu      sync.Mutex
	closed  bool
	entries map[string]*entry
	wg      sync.WaitGroup
}

type entry struct {
	orchestrator     *vowifi.Orchestrator
	busy             bool
	reconnectPending bool
	stopWatch        func()
}

func New(options Options) *Manager {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.OperationTimeout <= 0 {
		options.OperationTimeout = defaultOperationTimeout
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:              ctx,
		cancel:           cancel,
		logger:           options.Logger,
		operationTimeout: options.OperationTimeout,
		onState:          options.OnState,
		factory:          options.Factory,
		entries:          make(map[string]*entry),
	}
}

// Ensure registers a runtime for deviceID on demand. This keeps device
// configuration and runtime lifecycle in sync when a modem is added after the
// service has already started.
func (manager *Manager) Ensure(ctx context.Context, deviceID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return ErrClosed
	}
	if _, exists := manager.entries[deviceID]; exists {
		manager.mu.Unlock()
		return nil
	}
	if manager.factory == nil {
		manager.mu.Unlock()
		return ErrNotRegistered
	}

	// The factory is called while holding the manager lock so concurrent status
	// and enable requests cannot create duplicate runtimes for the same device.
	orchestrator, err := manager.factory(ctx, deviceID)
	if err != nil {
		manager.mu.Unlock()
		return err
	}
	if orchestrator == nil {
		manager.mu.Unlock()
		return errors.New("vowifi runtime: factory returned a nil orchestrator")
	}
	state := orchestrator.State()
	if state.DeviceID != deviceID {
		manager.mu.Unlock()
		_ = orchestrator.Close(context.Background())
		return fmt.Errorf(
			"vowifi runtime: factory returned device %q for %q",
			state.DeviceID,
			deviceID,
		)
	}
	states, stopWatch := orchestrator.Subscribe(8)
	manager.entries[deviceID] = &entry{
		orchestrator: orchestrator,
		stopWatch:    stopWatch,
	}
	manager.wg.Add(1)
	manager.mu.Unlock()

	go manager.watch(deviceID, states)
	return nil
}

func (manager *Manager) Register(orchestrator *vowifi.Orchestrator) error {
	if orchestrator == nil {
		return errors.New("vowifi runtime: orchestrator is nil")
	}
	state := orchestrator.State()
	if state.DeviceID == "" {
		return errors.New("vowifi runtime: orchestrator device ID is empty")
	}

	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return ErrClosed
	}
	if _, exists := manager.entries[state.DeviceID]; exists {
		manager.mu.Unlock()
		return fmt.Errorf("vowifi runtime: device %q is already registered", state.DeviceID)
	}
	states, stopWatch := orchestrator.Subscribe(8)
	item := &entry{
		orchestrator: orchestrator,
		stopWatch:    stopWatch,
	}
	manager.entries[state.DeviceID] = item
	manager.wg.Add(1)
	manager.mu.Unlock()

	go manager.watch(state.DeviceID, states)
	return nil
}

func (manager *Manager) State(deviceID string) (vowifi.State, error) {
	manager.mu.Lock()
	item := manager.entries[deviceID]
	closed := manager.closed
	manager.mu.Unlock()
	if item == nil {
		if closed {
			return vowifi.State{}, ErrClosed
		}
		if err := manager.Ensure(manager.ctx, deviceID); err != nil {
			return vowifi.State{}, err
		}
		manager.mu.Lock()
		item = manager.entries[deviceID]
		manager.mu.Unlock()
	}
	return item.orchestrator.State(), nil
}

// RequestEnabled queues an enable or disable transaction and returns
// immediately. Callers observe progress through State; provider errors are
// persisted in the orchestrator state instead of being lost with an HTTP
// request context.
func (manager *Manager) RequestEnabled(deviceID string, enabled bool) (vowifi.State, error) {
	if err := manager.Ensure(manager.ctx, deviceID); err != nil {
		return vowifi.State{}, err
	}
	return manager.startOperation(deviceID, false, func(ctx context.Context, orchestrator *vowifi.Orchestrator) error {
		if enabled {
			_, err := orchestrator.Enable(ctx)
			return err
		}
		_, err := orchestrator.Disable(ctx)
		return err
	})
}

func (manager *Manager) RequestReconnect(deviceID string) (vowifi.State, error) {
	if err := manager.Ensure(manager.ctx, deviceID); err != nil {
		return vowifi.State{}, err
	}
	return manager.startOperation(deviceID, true, func(ctx context.Context, orchestrator *vowifi.Orchestrator) error {
		_, err := orchestrator.Reconnect(ctx)
		return err
	})
}

func (manager *Manager) SendSMS(
	ctx context.Context,
	deviceID string,
	request vowifi.SMSSubmitRequest,
) (vowifi.SMSSubmitResult, error) {
	if err := manager.Ensure(ctx, deviceID); err != nil {
		return vowifi.SMSSubmitResult{}, err
	}
	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return vowifi.SMSSubmitResult{}, ErrClosed
	}
	item := manager.entries[deviceID]
	manager.mu.Unlock()
	if item == nil {
		return vowifi.SMSSubmitResult{}, ErrNotRegistered
	}
	return item.orchestrator.SendSMS(ctx, request)
}

func (manager *Manager) startOperation(
	deviceID string,
	coalesceReconnect bool,
	operation func(context.Context, *vowifi.Orchestrator) error,
) (vowifi.State, error) {
	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return vowifi.State{}, ErrClosed
	}
	item := manager.entries[deviceID]
	if item == nil {
		manager.mu.Unlock()
		return vowifi.State{}, ErrNotRegistered
	}
	if item.busy {
		state := item.orchestrator.State()
		if coalesceReconnect {
			// Route changes and repeated reconnect clicks only need the latest
			// result. Keep one pending reconnect behind the active lifecycle
			// operation instead of rejecting the request or running two modem/
			// tunnel transactions concurrently.
			item.reconnectPending = true
			manager.mu.Unlock()
			return state, nil
		}
		manager.mu.Unlock()
		return state, ErrOperationInProgress
	}
	item.busy = true
	manager.wg.Add(1)
	manager.mu.Unlock()

	go manager.runOperations(deviceID, item, operation)
	return item.orchestrator.State(), nil
}

func (manager *Manager) runOperations(
	deviceID string,
	item *entry,
	operation func(context.Context, *vowifi.Orchestrator) error,
) {
	defer manager.wg.Done()
	for {
		ctx, cancel := context.WithTimeout(manager.ctx, manager.operationTimeout)
		err := operation(ctx, item.orchestrator)
		cancel()
		if err != nil &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, vowifi.ErrAlreadyEnabled) {
			manager.logger.Warn(
				"VoWiFi operation failed",
				"device_id", deviceID,
				"error", err,
			)
		}
		manager.mu.Lock()
		if manager.closed || !item.reconnectPending {
			item.busy = false
			manager.mu.Unlock()
			return
		}
		item.reconnectPending = false
		manager.mu.Unlock()

		// Read the route only when this runs. If the user bound, unbound, then
		// rebound while busy, the single reconnect uses the final persisted
		// binding instead of replaying stale intermediate routes.
		operation = func(ctx context.Context, orchestrator *vowifi.Orchestrator) error {
			_, err := orchestrator.Reconnect(ctx)
			return err
		}
	}
}

func (manager *Manager) watch(deviceID string, states <-chan vowifi.State) {
	defer manager.wg.Done()
	for {
		select {
		case <-manager.ctx.Done():
			return
		case state, ok := <-states:
			if !ok {
				return
			}
			if manager.onState == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(manager.ctx, 5*time.Second)
			err := manager.onState(ctx, state)
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				manager.logger.Error(
					"persist VoWiFi state",
					"device_id", deviceID,
					"phase", state.Phase,
					"error", err,
				)
			}
		}
	}
}

func (manager *Manager) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return nil
	}
	manager.closed = true
	manager.cancel()
	items := make([]*entry, 0, len(manager.entries))
	for _, item := range manager.entries {
		items = append(items, item)
	}
	manager.mu.Unlock()

	var closeErrors []error
	for _, item := range items {
		if item.stopWatch != nil {
			item.stopWatch()
		}
		if err := item.orchestrator.Close(ctx); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}

	done := make(chan struct{})
	go func() {
		manager.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		closeErrors = append(closeErrors, ctx.Err())
	case <-done:
	}
	return errors.Join(closeErrors...)
}
