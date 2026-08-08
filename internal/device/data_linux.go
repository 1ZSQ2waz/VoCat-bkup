//go:build linux

package device

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"vocat/internal/modem"
)

func setQMINetwork(
	ctx context.Context,
	candidate modem.Candidate,
	enabled bool,
	apn string,
	ipVersion string,
) (NetworkResult, error) {
	qmiNetwork, err := exec.LookPath("qmi-network")
	if err != nil {
		return NetworkResult{}, fmt.Errorf("%w: install libqmi-utils to control %s", ErrDataBackendUnavailable, candidate.QMIControl)
	}
	profile, err := os.CreateTemp("", "vocat-qmi-*.conf")
	if err != nil {
		return NetworkResult{}, fmt.Errorf("create temporary QMI profile: %w", err)
	}
	profilePath := profile.Name()
	defer os.Remove(profilePath)
	ipType := map[string]string{"IP": "4", "IPV6": "6", "IPV4V6": "4"}[ipVersion]
	if _, err := fmt.Fprintf(profile, "APN=%s\nIP_TYPE=%s\nPROXY=yes\n", apn, ipType); err != nil {
		_ = profile.Close()
		return NetworkResult{}, fmt.Errorf("write temporary QMI profile: %w", err)
	}
	if err := profile.Chmod(0o600); err != nil {
		_ = profile.Close()
		return NetworkResult{}, fmt.Errorf("protect temporary QMI profile: %w", err)
	}
	if err := profile.Close(); err != nil {
		return NetworkResult{}, fmt.Errorf("close temporary QMI profile: %w", err)
	}

	action := "stop"
	if enabled {
		action = "start"
	}
	command := exec.CommandContext(ctx, qmiNetwork, "--profile="+profilePath, candidate.QMIControl, action)
	output, err := command.CombinedOutput()
	detail := strings.TrimSpace(string(output))
	if err != nil {
		lowerDetail := strings.ToLower(detail)
		idempotentStop := !enabled && (strings.Contains(lowerDetail, "already stopped") ||
			strings.Contains(lowerDetail, "not started") || strings.Contains(lowerDetail, "no network"))
		if !idempotentStop {
			return NetworkResult{}, fmt.Errorf("qmi-network %s failed: %w: %s", action, err, detail)
		}
	}
	if ipCommand, lookErr := exec.LookPath("ip"); lookErr == nil {
		linkAction := "down"
		if enabled {
			linkAction = "up"
		}
		linkOutput, linkErr := exec.CommandContext(ctx, ipCommand, "link", "set", "dev", candidate.NetworkInterface, linkAction).CombinedOutput()
		if linkErr != nil {
			return NetworkResult{}, fmt.Errorf("set %s %s: %w: %s", candidate.NetworkInterface, linkAction, linkErr, strings.TrimSpace(string(linkOutput)))
		}
	}
	if enabled {
		if busybox, lookErr := exec.LookPath("busybox"); lookErr == nil {
			dhcpOutput, dhcpErr := exec.CommandContext(ctx, busybox, "udhcpc", "-q", "-n", "-t", "5", "-T", "3", "-i", candidate.NetworkInterface).CombinedOutput()
			if dhcpErr != nil {
				rollbackCtx, cancelRollback := context.WithTimeout(context.Background(), managerCommandCleanupTimeout)
				defer cancelRollback()
				_, _ = exec.CommandContext(rollbackCtx, qmiNetwork, "--profile="+profilePath, candidate.QMIControl, "stop").CombinedOutput()
				if ipCommand, lookErr := exec.LookPath("ip"); lookErr == nil {
					_, _ = exec.CommandContext(rollbackCtx, ipCommand, "link", "set", "dev", candidate.NetworkInterface, "down").CombinedOutput()
				}
				return NetworkResult{}, fmt.Errorf("QMI session started but DHCP failed: %w: %s", dhcpErr, strings.TrimSpace(string(dhcpOutput)))
			}
			if value := strings.TrimSpace(string(dhcpOutput)); value != "" {
				detail = strings.TrimSpace(detail + "\n" + value)
			}
		}
	}
	return NetworkResult{
		Enabled:       enabled,
		Backend:       "qmi",
		Interface:     candidate.NetworkInterface,
		ControlDevice: candidate.QMIControl,
		APN:           apn,
		IPVersion:     ipVersion,
		Detail:        detail,
	}, nil
}

const managerCommandCleanupTimeout = 15 * time.Second
