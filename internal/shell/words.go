package shell

import (
	"encoding/binary"
	"io"
	"os"
	"strings"
	"unicode"
)

// ParseCd recognizes a plain "cd [dir]" command and returns dir without
// quotes; "cd" alone gives "~". A line the shell would have to expand or
// chain (variables, globs, ";", "&&", several arguments) is not a plain cd.
func ParseCd(line string) (dir string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "cd" {
		return "~", true
	}
	rest, found := strings.CutPrefix(line, "cd ")
	if !found {
		return "", false
	}
	return word(strings.TrimSpace(rest))
}

// special are the characters that make the shell do more than take a word
// as it is.
const special = " \t;&|<>()$`*?[{"

// word unquotes s if it is one shell word with nothing to expand.
func word(s string) (string, bool) {
	var b strings.Builder
	var quote rune
	escaped := false
	for _, r := range s {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				b.WriteRune(r)
			}
		case quote == '"':
			switch r {
			case '"':
				quote = 0
			case '\\':
				escaped = true
			case '$', '`':
				return "", false
			default:
				b.WriteRune(r)
			}
		case r == '\\':
			escaped = true
		case r == '\'' || r == '"':
			quote = r
		case strings.ContainsRune(special, r):
			return "", false
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 || escaped {
		return "", false
	}
	return b.String(), true
}

// Quote makes s safe to paste into a shell command: plain names stay as
// they are, others go in single quotes.
func Quote(s string) string {
	plain := s != ""
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("._-+/@%:,=", r) {
			plain = false
			break
		}
	}
	if plain {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Runnable reports whether path is a program: an executable regular file
// that starts with "#!" or is a Mach-O binary. The x bit alone is not
// enough: on exFAT and FAT drives every file has it.
func Runnable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var b [4]byte
	n, _ := io.ReadFull(f, b[:])
	if n >= 2 && b[0] == '#' && b[1] == '!' {
		return true
	}
	if n < 4 {
		return false
	}
	switch binary.BigEndian.Uint32(b[:]) {
	case 0xfeedface, 0xfeedfacf, 0xcefaedfe, 0xcffaedfe, 0xcafebabe: // Mach-O 32/64 both byte orders, universal
		return true
	}
	return false
}
