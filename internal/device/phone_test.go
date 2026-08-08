package device

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vocat/internal/modem"
)

func TestReadPhoneNumberFallsBackToOwnNumbersAndRestoresPhonebook(t *testing.T) {
	client := &transcriptClient{steps: []clientStep{
		{command: "AT+CNUM", response: okResponse()},
		{command: "AT+CPBS?", response: okResponse(`+CPBS: "SM",0,250`)},
		{command: `AT+CPBS="ON"`, response: okResponse()},
		{command: "AT+CPBR=?", response: okResponse("+CPBR: (1-3),40,20")},
		{command: "AT+CPBR=1", response: okResponse(`+CPBR: 1,"",129,""`)},
		{
			command:  "AT+CPBR=2",
			response: okResponse(`+CPBR: 2,"+44 7700 900123",145,"Own"`),
		},
		{
			command: `AT+CPBS="SM"`,
			err:     errors.New("restore failed"),
		},
	}}
	manager, err := NewManager(Options{})
	if err != nil {
		t.Fatal(err)
	}

	phone, warnings := manager.readPhoneNumber(context.Background(), client)
	if phone.Number != "+447700900123" || phone.Source != PhoneSourceOwnNumber {
		t.Fatalf("phone = %#v", phone)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "restore phonebook") {
		t.Fatalf("warnings = %#v", warnings)
	}
	client.assertDone(t)
}

func TestReadPhoneNumberFallsBackToEFMSISDN(t *testing.T) {
	record := strings.Repeat("FF", 18) +
		"0791447700091032FFFFFFFFFFFF"
	client := &transcriptClient{steps: []clientStep{
		{command: "AT+CNUM", response: okResponse()},
		{command: "AT+CPBS?", response: okResponse(`+CPBS: "SM",0,250`)},
		{
			command: `AT+CPBS="ON"`,
			err:     errors.New("Own Numbers unavailable"),
		},
		{
			command: "AT+CRSM=192,28480,0,0,0",
			err:     errors.New("zero-length GET RESPONSE unsupported"),
		},
		{
			command:  "AT+CRSM=192,28480,0,0,15",
			response: okResponse(`+CRSM: 144,0,"62198205422100200283026F408A01"`),
		},
		{
			command:  "AT+CRSM=178,28480,1,4,32",
			response: okResponse(`+CRSM: 144,0,"` + record + `"`),
		},
	}}
	manager, err := NewManager(Options{})
	if err != nil {
		t.Fatal(err)
	}

	phone, warnings := manager.readPhoneNumber(context.Background(), client)
	if phone.Number != "+447700900123" || phone.Source != PhoneSourceEFMSISDN {
		t.Fatalf("phone = %#v; warnings = %#v", phone, warnings)
	}
	client.assertDone(t)
}

func TestPhoneNumberIsNeverDerivedFromSubscriberIdentifiers(t *testing.T) {
	if got := parsePhoneResponse(
		modem.Response{Lines: []string{"+CNUM: ,,,"}},
		"+CNUM:",
	); got != "" {
		t.Fatalf("empty CNUM parsed as %q", got)
	}
	if got := normalizePhoneNumber("460001234567890/89860012345678901234"); got != "" {
		t.Fatalf("invalid combined subscriber identifiers parsed as %q", got)
	}
}
