package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"vocat/internal/i18n"
)

type ProbeResult struct {
	Reachable      bool   `json:"reachable"`
	HandshakeOK    bool   `json:"handshake_ok"`
	UDPAssociateOK bool   `json:"udp_associate_ok"`
	AuthMethod     string `json:"auth_method,omitempty"`
	RelayAddr      string `json:"relay_addr,omitempty"`
	Diagnosis      string `json:"diagnosis,omitempty"`
	Hint           string `json:"hint,omitempty"`
}

func ProbeSOCKS5(
	ctx context.Context,
	address string,
	username string,
	password string,
	timeout time.Duration,
) (ProbeResult, error) {
	address = strings.TrimSpace(address)
	if _, _, err := net.SplitHostPort(address); err != nil {
		return ProbeResult{}, fmt.Errorf("proxy: upstream address must be host:port: %w", err)
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	probeContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	connection, err := (&net.Dialer{Timeout: timeout}).DialContext(probeContext, "tcp", address)
	if err != nil {
		return ProbeResult{
			Diagnosis: "tcp_unreachable",
			Hint:      i18n.T("检查地址、端口、防火墙与上游代理监听状态。"),
		}, err
	}
	defer connection.Close()
	result := ProbeResult{Reachable: true}
	_ = connection.SetDeadline(time.Now().Add(timeout))

	methods := []byte{0}
	if username != "" {
		methods = append(methods, 2)
	}
	greeting := append([]byte{5, byte(len(methods))}, methods...)
	if _, err := connection.Write(greeting); err != nil {
		return result, err
	}
	methodResponse := make([]byte, 2)
	if _, err := io.ReadFull(connection, methodResponse); err != nil {
		return result, err
	}
	if methodResponse[0] != 5 || methodResponse[1] == 0xff {
		result.Diagnosis = "no_acceptable_auth"
		return result, errors.New("proxy: upstream rejected all SOCKS5 authentication methods")
	}
	switch methodResponse[1] {
	case 0:
		result.AuthMethod = "none"
	case 2:
		result.AuthMethod = "username_password"
		if username == "" || len(username) > 255 || len(password) > 255 {
			return result, errors.New("proxy: upstream requires username/password authentication")
		}
		authRequest := []byte{1, byte(len(username))}
		authRequest = append(authRequest, []byte(username)...)
		authRequest = append(authRequest, byte(len(password)))
		authRequest = append(authRequest, []byte(password)...)
		if _, err := connection.Write(authRequest); err != nil {
			return result, err
		}
		authResponse := make([]byte, 2)
		if _, err := io.ReadFull(connection, authResponse); err != nil {
			return result, err
		}
		if authResponse[0] != 1 || authResponse[1] != 0 {
			result.Diagnosis = "authentication_failed"
			return result, errors.New("proxy: upstream username/password authentication failed")
		}
	default:
		result.AuthMethod = fmt.Sprintf("method_%d", methodResponse[1])
		return result, errors.New("proxy: upstream selected an unsupported authentication method")
	}
	result.HandshakeOK = true

	if _, err := connection.Write([]byte{5, 3, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		return result, err
	}
	reader := bufio.NewReader(connection)
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return result, err
	}
	if header[0] != 5 {
		return result, errors.New("proxy: invalid UDP ASSOCIATE response version")
	}
	if header[1] != 0 {
		result.Diagnosis = "udp_associate_rejected"
		result.Hint = i18n.T("该代理不能承载 ePDG 所需的 UDP；启用上游 SOCKS5 UDP 转发后重试。")
		return result, fmt.Errorf("proxy: upstream rejected UDP ASSOCIATE with code %d", header[1])
	}
	host, err := readSOCKSAddress(reader, header[3])
	if err != nil {
		return result, err
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return result, err
	}
	port := int(portBytes[0])<<8 | int(portBytes[1])
	result.UDPAssociateOK = true
	result.RelayAddr = net.JoinHostPort(host, fmt.Sprintf("%d", port))
	result.Diagnosis = "ready"
	result.Hint = i18n.T("TCP 握手、认证和 UDP ASSOCIATE 均通过。")
	return result, nil
}

func readSOCKSAddress(reader io.Reader, addressType byte) (string, error) {
	switch addressType {
	case 1:
		value := make([]byte, net.IPv4len)
		if _, err := io.ReadFull(reader, value); err != nil {
			return "", err
		}
		return net.IP(value).String(), nil
	case 3:
		var length [1]byte
		if _, err := io.ReadFull(reader, length[:]); err != nil {
			return "", err
		}
		if length[0] == 0 {
			return "", errors.New("empty SOCKS5 domain")
		}
		value := make([]byte, int(length[0]))
		if _, err := io.ReadFull(reader, value); err != nil {
			return "", err
		}
		return string(value), nil
	case 4:
		value := make([]byte, net.IPv6len)
		if _, err := io.ReadFull(reader, value); err != nil {
			return "", err
		}
		return net.IP(value).String(), nil
	default:
		return "", errors.New("unsupported SOCKS5 address type")
	}
}
