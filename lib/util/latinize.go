package util

import "strings"

func Latinize(str string) string {
	s := strings.Builder{}
	for _, r := range str {
		s.WriteRune(latinizeRune(r))
	}
	return s.String()
}

func latinizeRune(c rune) rune {
	switch c {
	case 'a':
		return 'ᴀ'
	case 'b':
		return 'ʙ'
	case 'c':
		return 'ᴄ'
	case 'd':
		return 'ᴅ'
	case 'e':
		return 'ᴇ'
	case 'f':
		return 'ғ'
	case 'g':
		return 'ɢ'
	case 'h':
		return 'ʜ'
	case 'i':
		return 'ɪ'
	case 'j':
		return 'ᴊ'
	case 'k':
		return 'ᴋ'
	case 'l':
		return 'ʟ'
	case 'm':
		return 'ᴍ'
	case 'n':
		return 'ɴ'
	case 'o':
		return 'ᴏ'
	case 'p':
		return 'ᴘ'
	case 'q':
		return 'q'
	case 'r':
		return 'ʀ'
	case 's':
		return 's'
	case 't':
		return 'ᴛ'
	case 'u':
		return 'ᴜ'
	case 'v':
		return 'ᴠ'
	case 'w':
		return 'ᴡ'
	case 'x':
		return 'x'
	case 'y':
		return 'ʏ'
	case 'z':
		return 'z'
	default:
		return c
	}
}
