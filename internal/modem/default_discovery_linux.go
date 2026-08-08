//go:build linux

package modem

func NewSystemDiscoverer() Discoverer {
	return NewSysFSDiscoverer("/sys", "/dev")
}
