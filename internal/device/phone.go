package device

import (
	"context"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vocat/internal/modem"
)

var phonePattern = regexp.MustCompile(`\+?[0-9][0-9 ()-]{3,}[0-9]`)

func (manager *Manager) readPhoneNumber(
	ctx context.Context,
	client modem.Client,
) (PhoneNumber, []string) {
	var warnings []string
	if response, err := manager.command(ctx, client, "AT+CNUM"); err == nil {
		if number := parsePhoneResponse(response, "+CNUM:"); number != "" {
			return PhoneNumber{
				Number: number,
				Source: PhoneSourceCNUM,
				Status: "号码来自模块/SIM 的 CNUM 记录",
			}, warnings
		}
	} else {
		warnings = append(warnings, "read CNUM: "+err.Error())
	}

	phonebook, phonebookWarnings := manager.readOwnNumberPhonebook(ctx, client)
	warnings = append(warnings, phonebookWarnings...)
	if phonebook != "" {
		return PhoneNumber{
			Number: phonebook,
			Source: PhoneSourceOwnNumber,
			Status: "号码来自 SIM Own Numbers 电话簿",
		}, warnings
	}

	raw, rawWarnings := manager.readEFMSISDN(ctx, client)
	warnings = append(warnings, rawWarnings...)
	if raw != "" {
		return PhoneNumber{
			Number: raw,
			Source: PhoneSourceEFMSISDN,
			Status: "号码来自 USIM EF_MSISDN 只读记录",
		}, warnings
	}
	return PhoneNumber{
		Status: "CNUM、Own Numbers 与 EF_MSISDN 均为空；号码不能由 IMSI/ICCID 推导，需要运营商或 IMS/VoWiFi 注册侧提供",
	}, warnings
}

func (manager *Manager) readOwnNumberPhonebook(
	ctx context.Context,
	client modem.Client,
) (number string, warnings []string) {
	previous := ""
	if response, err := manager.command(ctx, client, "AT+CPBS?"); err == nil {
		previous = parseSelectedPhonebook(response)
	} else {
		warnings = append(warnings, "query current phonebook: "+err.Error())
	}
	response, err := manager.command(ctx, client, `AT+CPBS="ON"`)
	if err != nil || !response.OK() {
		if err != nil {
			warnings = append(warnings, "select Own Numbers phonebook: "+err.Error())
		}
		return "", warnings
	}
	if previous != "" && previous != "ON" {
		defer func() {
			restoreCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if _, restoreErr := manager.command(
				restoreCtx,
				client,
				fmt.Sprintf(`AT+CPBS="%s"`, previous),
			); restoreErr != nil {
				warnings = append(warnings, "restore phonebook: "+restoreErr.Error())
			}
		}()
	}

	start, end := 1, 2
	if response, rangeErr := manager.command(ctx, client, "AT+CPBR=?"); rangeErr == nil {
		if parsedStart, parsedEnd, ok := parsePhonebookRange(response); ok {
			start, end = parsedStart, parsedEnd
		}
	} else {
		warnings = append(warnings, "query Own Numbers range: "+rangeErr.Error())
	}
	if end > start+9 {
		end = start + 9
	}
	for index := start; index <= end; index++ {
		response, readErr := manager.command(ctx, client, fmt.Sprintf("AT+CPBR=%d", index))
		if readErr != nil {
			continue
		}
		if number := parsePhoneResponse(response, "+CPBR:"); number != "" {
			return number, warnings
		}
	}
	return "", warnings
}

func parseSelectedPhonebook(response modem.Response) string {
	value := valueAfterPrefix(response, "+CPBS:")
	values := csvValues(value)
	if len(values) == 0 {
		return ""
	}
	selected := strings.ToUpper(strings.Trim(values[0], `"`))
	if len(selected) < 1 || len(selected) > 4 {
		return ""
	}
	for _, character := range selected {
		if character < 'A' || character > 'Z' {
			return ""
		}
	}
	return selected
}

func parsePhonebookRange(response modem.Response) (int, int, bool) {
	pattern := regexp.MustCompile(`\((\d+)-(\d+)\)`)
	for _, line := range response.Lines {
		match := pattern.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		start, startErr := strconv.Atoi(match[1])
		end, endErr := strconv.Atoi(match[2])
		if startErr == nil && endErr == nil && start > 0 && end >= start {
			return start, end, true
		}
	}
	return 0, 0, false
}

func parsePhoneResponse(response modem.Response, prefix string) string {
	for _, line := range response.Lines {
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), strings.ToUpper(prefix)) {
			continue
		}
		for _, match := range phonePattern.FindAllString(line, -1) {
			if number := normalizePhoneNumber(match); number != "" {
				return number
			}
		}
	}
	return ""
}

func normalizePhoneNumber(value string) string {
	var result strings.Builder
	for _, character := range strings.TrimSpace(value) {
		switch {
		case character >= '0' && character <= '9':
			result.WriteRune(character)
		case character == '+' && result.Len() == 0:
			result.WriteRune(character)
		case character == ' ' || character == '-' || character == '(' || character == ')':
		default:
			return ""
		}
	}
	number := result.String()
	digits := strings.TrimPrefix(number, "+")
	if len(digits) < 5 || len(digits) > 20 {
		return ""
	}
	return number
}

type crsmPath struct {
	value string
}

func (manager *Manager) readEFMSISDN(
	ctx context.Context,
	client modem.Client,
) (string, []string) {
	var warnings []string
	for _, path := range []crsmPath{{}, {"3F007FFF"}, {"3F007F10"}} {
		recordLength, recordCount, ok := 0, 0, false
		for _, responseLength := range []int{0, 15} {
			status, err := manager.command(
				ctx,
				client,
				crsmCommand(192, 0, responseLength, path.value),
			)
			if err != nil {
				warnings = append(
					warnings,
					fmt.Sprintf(
						"read EF_MSISDN metadata (path %q, length %d): %v",
						path.value,
						responseLength,
						err,
					),
				)
				continue
			}
			recordLength, recordCount, ok = msisdnRecordShape(status)
			if ok {
				break
			}
		}
		if !ok {
			continue
		}
		if recordCount > 10 {
			recordCount = 10
		}
		for index := 1; index <= recordCount; index++ {
			record, readErr := manager.command(
				ctx,
				client,
				crsmCommand(178, index, recordLength, path.value),
			)
			if readErr != nil {
				warnings = append(
					warnings,
					fmt.Sprintf(
						"read EF_MSISDN record %d (path %q): %v",
						index,
						path.value,
						readErr,
					),
				)
				continue
			}
			if number := decodeMSISDNRecord(record); number != "" {
				return number, warnings
			}
		}
	}
	return "", warnings
}

func crsmCommand(operation, index, length int, path string) string {
	p2 := 0
	if operation == 178 {
		p2 = 4
	}
	command := fmt.Sprintf("AT+CRSM=%d,28480,%d,%d,%d", operation, index, p2, length)
	if path != "" {
		command += fmt.Sprintf(`,"","%s"`, path)
	}
	return command
}

func crsmPayload(response modem.Response) []byte {
	value := valueAfterPrefix(response, "+CRSM:")
	values := csvValues(value)
	if len(values) < 3 {
		return nil
	}
	sw1, sw1Err := strconv.Atoi(values[0])
	sw2, sw2Err := strconv.Atoi(values[1])
	if sw1Err != nil || sw2Err != nil ||
		!((sw1 == 144 && sw2 == 0) || sw1 == 145) {
		return nil
	}
	payload := strings.Trim(values[2], `" `)
	decoded, err := hex.DecodeString(payload)
	if err != nil {
		return nil
	}
	return decoded
}

func msisdnRecordShape(response modem.Response) (recordLength, recordCount int, ok bool) {
	payload := crsmPayload(response)
	for index := 0; index+1 < len(payload); index++ {
		tag := payload[index]
		length := int(payload[index+1])
		start := index + 2
		end := start + length
		if end > len(payload) {
			continue
		}
		if tag == 0x82 && length >= 5 {
			recordLength = int(payload[end-3])<<8 | int(payload[end-2])
			recordCount = int(payload[end-1])
			if recordLength >= 14 && recordLength <= 255 && recordCount > 0 {
				return recordLength, recordCount, true
			}
		}
	}
	if len(payload) >= 15 &&
		payload[4] == 0x6f && payload[5] == 0x40 &&
		payload[6] == 0x04 &&
		(payload[13] == 0x01 || payload[13] == 0x03) {
		fileSize := int(payload[2])<<8 | int(payload[3])
		recordLength = int(payload[14])
		if recordLength >= 14 && fileSize >= recordLength {
			return recordLength, fileSize / recordLength, true
		}
	}
	return 0, 0, false
}

func decodeMSISDNRecord(response modem.Response) string {
	payload := crsmPayload(response)
	if len(payload) < 14 {
		return ""
	}
	footer := payload[len(payload)-14:]
	storedLength := int(footer[0])
	if storedLength < 2 || storedLength == 0xff {
		return ""
	}
	bcdLength := storedLength - 1
	if bcdLength > 10 {
		bcdLength = 10
	}
	var result strings.Builder
	if footer[1]&0x70 == 0x10 {
		result.WriteByte('+')
	}
	for _, value := range footer[2 : 2+bcdLength] {
		for _, digit := range []byte{value & 0x0f, value >> 4} {
			if digit <= 9 {
				result.WriteByte('0' + digit)
			}
		}
	}
	return normalizePhoneNumber(result.String())
}
