package converter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/huaky-tec-fornb/go-net-tool/internal/model"
)

// HexDump formats raw bytes into Wireshark-style hex+ASCII dump.
// Each line: 8-digit offset, 16 bytes as hex pairs (grouped 8+8), ASCII representation.
func HexDump(data []byte, offsetPrefix bool, offsetStart uint64) string {
	if len(data) == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		line := data[i:end]

		if offsetPrefix {
			b.WriteString(fmt.Sprintf("%08X  ", offsetStart+uint64(i)))
		}

		// Hex pairs
		hexPart := make([]string, len(line))
		for j, bt := range line {
			hexPart[j] = fmt.Sprintf("%02X", bt)
		}
		hexStr := strings.Join(hexPart, " ")
		// Add extra space between byte 8 and 9
		if len(line) > 8 {
			idx := 0
			for k := 0; k < 8; k++ {
				idx += 3 // "XX " per byte
			}
			hexStr = hexStr[:idx-1] + "  " + hexStr[idx:]
		}
		b.WriteString(hexStr)

		// Pad short lines for alignment
		if len(line) < 16 {
			padLen := (16-len(line))*3 + 1
			if len(line) <= 8 {
				padLen += 1
			}
			b.WriteString(strings.Repeat(" ", padLen))
		}

		// ASCII representation
		b.WriteString("  ")
		for _, bt := range line {
			if bt >= 0x20 && bt < 0x7f {
				b.WriteByte(bt)
			} else {
				b.WriteByte('.')
			}
		}
		if i+16 < len(data) {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// BytesToEscapedText converts bytes to a display-safe string.
// Non-printable bytes become '.', CR/LF/TAB are preserved.
func BytesToEscapedText(data []byte) string {
	var b strings.Builder
	for _, bt := range data {
		r := rune(bt)
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(bt)
		} else if unicode.IsPrint(r) {
			b.WriteByte(bt)
		} else {
			b.WriteByte('.')
		}
	}
	return b.String()
}

// FormatMessage formats a Message's data for display based on mode.
func FormatMessage(msg model.Message, mode string) string {
	if mode == "hex" {
		return HexDump(msg.Data, true, 0)
	}
	return BytesToEscapedText(msg.Data)
}
