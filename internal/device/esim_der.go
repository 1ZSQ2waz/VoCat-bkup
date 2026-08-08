package device

import "fmt"

// DER/BER-TLV encoding helpers — the encoder counterpart to the decoder
// (derParse/derDecodeOne) in esim.go. These build the ES10 request bodies the
// eUICC consumes (AuthenticateServer, PrepareDownload, LoadBoundProfilePackage,
// …), whose tags mix one-byte (0x30, 0x04) and two-byte (0x5F37, 0xBF38) forms
// and whose payloads can exceed the 0x80 short-length threshold.

// derEncodeTag emits a tag's identifier bytes (1 byte for tags < 0x100, more for
// long-form tags such as 0x5F37 or 0xBF38).
func derEncodeTag(tag int) []byte {
	if tag < 0x100 {
		return []byte{byte(tag)}
	}
	out := make([]byte, 0, 3)
	started := false
	for shift := 24; shift >= 0; shift -= 8 {
		b := byte(tag >> shift)
		if b != 0 || started {
			out = append(out, b)
			started = true
		}
	}
	return out
}

// derEncodeLength emits a length in short form (< 0x80) or long form (0x81/0x82/0x83).
func derEncodeLength(length int) []byte {
	switch {
	case length < 0x80:
		return []byte{byte(length)}
	case length < 0x100:
		return []byte{0x81, byte(length)}
	case length < 0x10000:
		return []byte{0x82, byte(length >> 8), byte(length)}
	default:
		return []byte{0x83, byte(length >> 16), byte(length >> 8), byte(length)}
	}
}

// derEncode builds one complete BER-TLV element: tag + length + value.
func derEncode(tag int, value []byte) []byte {
	out := derEncodeTag(tag)
	out = append(out, derEncodeLength(len(value))...)
	return append(out, value...)
}

// derConstruct builds a constructed element whose value is the concatenation of
// already-encoded child elements.
func derConstruct(tag int, children ...[]byte) []byte {
	var value []byte
	for _, child := range children {
		value = append(value, child...)
	}
	return derEncode(tag, value)
}

// derFindValue parses data and returns the value of the first node with tag,
// searching recursively. Use this (not derValue) when the target may be nested
// inside an enclosing element — e.g. transactionId (0x80) inside serverSigned1.
func derFindValue(data []byte, tag int) []byte {
	nodes := derFindAll(derParse(data), tag)
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0].value
}

// derElementAt decodes the single BER-TLV element starting at buf[offset] and
// reports its tag, the number of header bytes (tag + length), and the total
// element length (header + value). It performs no recursion, so callers can walk
// a buffer by explicit offset — exactly what LoadBoundProfilePackage segmentation
// needs to slice the package at TLV boundaries.
func derElementAt(buf []byte, offset int) (tag int, headerLen int, totalLen int, err error) {
	index := offset
	if index >= len(buf) {
		return 0, 0, 0, fmt.Errorf("esim: element at %d out of range", offset)
	}
	first := buf[index]
	index++
	tag = int(first)
	if first&0x1F == 0x1F { // long-form tag
		for index < len(buf) {
			b := buf[index]
			index++
			tag = tag<<8 | int(b)
			if b&0x80 == 0 {
				break
			}
		}
	}
	if index >= len(buf) {
		return 0, 0, 0, fmt.Errorf("esim: truncated tag at %d", offset)
	}
	lengthByte := buf[index]
	index++
	length := 0
	if lengthByte&0x80 == 0 {
		length = int(lengthByte)
	} else {
		count := int(lengthByte & 0x7F)
		if count == 0 || count > 4 || index+count > len(buf) {
			return 0, 0, 0, fmt.Errorf("esim: bad length at %d", offset)
		}
		for i := 0; i < count; i++ {
			length = length<<8 | int(buf[index])
			index++
		}
	}
	headerLen = index - offset
	if headerLen+length > len(buf)-offset {
		return 0, 0, 0, fmt.Errorf("esim: element at %d overruns buffer", offset)
	}
	return tag, headerLen, headerLen + length, nil
}
