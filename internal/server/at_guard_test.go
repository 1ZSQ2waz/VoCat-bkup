package server

import "testing"

func TestValidateATCommandBlocksTrafficMessagingAndDialActions(t *testing.T) {
	t.Parallel()
	for _, command := range []string{
		"AT+CGATT=1",
		"AT+CGACT=1,1",
		"AT+CGDATA=\"PPP\",1",
		"AT+QNETDEVCTL=1,1,1",
		"AT+QIACT=1",
		"AT+CMGS=42",
		"AT+CMSS=7",
		"AT+CMGC=12",
		"AT+QCMGS=42",
		"AT+CUSD=1,\"*100#\"",
		"ATD12345;",
		"ATA",
		"ATH",
		"AT+CSQ; +CGACT = 1,1",
		"AT+CSQ;+CMSS=7",
		"AT+CSQ;D12345;",
	} {
		if err := validateATCommand(command); err == nil {
			t.Errorf("validateATCommand(%q) permitted a guarded mutation", command)
		}
	}
}

func TestValidateATCommandAllowsReadOnlyStatusQueries(t *testing.T) {
	t.Parallel()
	for _, command := range []string{
		"AT",
		"AT+CPIN?",
		"AT+CGATT?",
		"AT+CGACT?",
		"AT+CFUN?",
		"AT+CIMI",
		"AT+CCID",
	} {
		if err := validateATCommand(command); err != nil {
			t.Errorf("validateATCommand(%q): %v", command, err)
		}
	}
}
