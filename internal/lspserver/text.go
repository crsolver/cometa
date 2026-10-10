package lspserver

import ()

func runeIndexAtUTF16(line string, column int) int {
	units, index := 0, 0
	for _, r := range line {
		width := 1
		if r > 0xffff {
			width = 2
		}
		if units+width > column {
			break
		}
		units += width
		index++
	}
	return index
}

func isIdentifierRune(value rune) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value > 127
}

func utf16Prefix(value string, units int) string {
	if units <= 0 {
		return ""
	}
	used := 0
	for byteIndex, r := range value {
		width := 1
		if r > 0xFFFF {
			width = 2
		}
		if used+width > units {
			return value[:byteIndex]
		}
		used += width
		if used == units {
			return value[:byteIndex+len(string(r))]
		}
	}
	return value
}

func utf16Length(value string) int {
	length := 0
	for _, r := range value {
		if r > 0xFFFF {
			length += 2
		} else {
			length++
		}
	}
	return length
}
