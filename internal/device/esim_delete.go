package device

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	// These errors mirror the standardized SGP.22 DeleteProfileResponse values.
	// Keep them exported so the HTTP layer can return actionable API errors
	// instead of leaking raw BER-TLV response bytes to the UI.
	ErrESIMDeleteProfileNotFound    = errors.New("esim: profile was not found on the eUICC")
	ErrESIMDeleteProfileNotDisabled = errors.New("esim: the active profile cannot be deleted; enable another profile first")
	ErrESIMDeleteDisallowedByPolicy = errors.New("esim: profile deletion is not allowed by its policy")
)

// EsimDeleteResult reports storage reclaimed by a successful ES10c delete.
type EsimDeleteResult struct {
	SpaceDelta int64
	Warning    string
}

func buildDeleteProfileRequest(iccid string) ([]byte, error) {
	bcd, err := encodeICCID(strings.TrimSpace(iccid))
	if err != nil {
		return nil, err
	}
	// SGP.22 ES10c DeleteProfileRequest: BF33 { 5A <ICCID BCD> }.
	return derConstruct(0xBF33, derEncode(0x5A, bcd)), nil
}

func deleteProfileResult(payload []byte) (byte, bool) {
	nodes := derParse(payload)
	if len(nodes) != 1 || nodes[0].tag != 0xBF33 {
		return 0, false
	}
	result := derFindValue(payload, 0x80)
	if len(result) != 1 {
		return 0, false
	}
	return result[0], true
}

func deleteProfileResponseError(result byte, payload []byte) error {
	raw := strings.ToUpper(hex.EncodeToString(payload))
	switch result {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("%w (result=0x%02X, raw %s)", ErrESIMDeleteProfileNotFound, result, raw)
	case 2:
		return fmt.Errorf("%w (result=0x%02X, raw %s)", ErrESIMDeleteProfileNotDisabled, result, raw)
	case 3:
		return fmt.Errorf("%w (result=0x%02X, raw %s)", ErrESIMDeleteDisallowedByPolicy, result, raw)
	default:
		return fmt.Errorf("esim: eUICC rejected DeleteProfile, result=0x%02X (raw %s)", result, raw)
	}
}

// ESIMDeleteProfile removes one installed, disabled profile through ES10c.
// Root CI certificates are not involved in local profile management; the
// eUICC authorizes this operation through its ISD-R interface.
func (manager *Manager) ESIMDeleteProfile(ctx context.Context, id, iccid, aidHex string) (*EsimDeleteResult, error) {
	request, err := buildDeleteProfileRequest(iccid)
	if err != nil {
		return nil, err
	}
	manager.esimMu.Lock()
	defer manager.esimMu.Unlock()
	if err := manager.waitForESIMRecovery(ctx, id); err != nil {
		return nil, err
	}
	channel, err := manager.openEuiccAID(ctx, id, targetEuiccAID(aidHex))
	if err != nil {
		return nil, err
	}
	defer channel.close(context.Background())

	freeBefore, beforeKnown := 0, false
	if info2, infoErr := channel.getEUICCInfo2(ctx); infoErr == nil {
		freeBefore, beforeKnown = euiccFreeNVRAM(info2)
	}

	// DeleteProfile is non-idempotent. Once submitted, finish reading the card's
	// result even if the browser request is cancelled.
	commitContext, cancelCommit := context.WithTimeout(context.WithoutCancel(ctx), csimAPDUTimeout)
	payload, err := channel.es10(commitContext, request)
	cancelCommit()
	if err != nil {
		return nil, err
	}
	result, ok := deleteProfileResult(payload)
	if !ok {
		return nil, fmt.Errorf("esim: unexpected DeleteProfile response %s", strings.ToUpper(hex.EncodeToString(payload)))
	}
	if err := deleteProfileResponseError(result, payload); err != nil {
		return nil, err
	}

	deleted := &EsimDeleteResult{}
	if info2, infoErr := channel.getEUICCInfo2(ctx); infoErr == nil {
		if freeAfter, afterKnown := euiccFreeNVRAM(info2); beforeKnown && afterKnown && freeAfter >= freeBefore {
			deleted.SpaceDelta = int64(freeAfter - freeBefore)
		}
	} else {
		deleted.Warning = "Profile was deleted, but reclaimed storage could not be read"
	}
	manager.removeCachedProfile(id, strings.TrimSpace(iccid))
	return deleted, nil
}
