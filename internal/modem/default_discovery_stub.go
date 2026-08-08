//go:build !linux

package modem

func NewSystemDiscoverer() Discoverer {
	return unsupportedDiscoverer{}
}
