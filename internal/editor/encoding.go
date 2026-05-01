package editor

import (
	"encoding/binary"
	"unicode/utf16"
)

// decodeToUTF8 detects the encoding of raw file bytes and returns a UTF-8 string.
// It handles UTF-8 (with or without BOM), UTF-16 LE, and UTF-16 BE.
func decodeToUTF8(data []byte) string {
	if len(data) >= 2 {
		// UTF-16 LE BOM: FF FE
		if data[0] == 0xFF && data[1] == 0xFE {
			return decodeUTF16(data[2:], binary.LittleEndian)
		}
		// UTF-16 BE BOM: FE FF
		if data[0] == 0xFE && data[1] == 0xFF {
			return decodeUTF16(data[2:], binary.BigEndian)
		}
	}

	// UTF-8 BOM: EF BB BF — strip it
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// Heuristic: if no BOM but content looks like UTF-16 LE (high ratio of
	// null bytes in odd positions for ASCII text), decode as UTF-16 LE.
	if len(data) >= 4 && looksLikeUTF16LE(data) {
		return decodeUTF16(data, binary.LittleEndian)
	}

	return string(data)
}

// looksLikeUTF16LE checks if the byte content appears to be UTF-16 LE encoded
// by examining whether every other byte is null (common for ASCII/Latin text).
func looksLikeUTF16LE(data []byte) bool {
	if len(data) < 4 || len(data)%2 != 0 {
		return false
	}
	// Check first 100 code units (or all if shorter)
	check := len(data)
	if check > 200 {
		check = 200
	}
	nulls := 0
	for i := 1; i < check; i += 2 {
		if data[i] == 0 {
			nulls++
		}
	}
	// If >80% of high bytes are null, likely UTF-16 LE
	return nulls > (check/2)*8/10
}

// decodeUTF16 converts UTF-16 encoded bytes to a UTF-8 Go string.
func decodeUTF16(data []byte, order binary.ByteOrder) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1] // drop trailing byte
	}

	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = order.Uint16(data[i*2:])
	}

	runes := utf16.Decode(u16s)
	return string(runes)
}
