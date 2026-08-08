package modem

import "testing"

func TestSelectATPortPrefersTTYUSB2AcrossUSBCompositions(t *testing.T) {
	ports := []Port{
		{Name: "ttyUSB2", InterfaceNumber: 0x02, Role: PortRoleDiagnostic},
		{Name: "ttyUSB3", InterfaceNumber: 0x05, Role: PortRoleModem},
	}
	selected := selectATPort(ports)
	if selected.Name != "ttyUSB2" || selected.InterfaceNumber != 0x02 {
		t.Fatalf("selected %#v, want ttyUSB2 on interface 02", selected)
	}
}
