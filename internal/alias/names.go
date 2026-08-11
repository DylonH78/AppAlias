package alias

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

var reservedNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

// Key is the case-insensitive identity Windows uses for an alias file.
func Key(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func Validate(name string) error {
	if name = strings.TrimSpace(name); name == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if len([]rune(name)) > 80 {
		return fmt.Errorf("alias must be 80 characters or fewer")
	}
	if name == "." || name == ".." || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("alias cannot end with a period or space")
	}
	for _, r := range name {
		if r < 32 || strings.ContainsRune(`<>:"/\\|?*`, r) {
			return fmt.Errorf("alias contains a character not allowed in Windows file names")
		}
	}
	base := strings.ToUpper(strings.Split(name, ".")[0])
	if _, found := reservedNames[base]; found {
		return fmt.Errorf("%q is a reserved Windows device name", name)
	}
	return nil
}

// Suggestions returns stable, de-duplicated names in preference order.
func Suggestions(displayName, executablePath string) []string {
	values := make([]string, 0, 4)
	if stem := strings.TrimSuffix(filepath.Base(executablePath), filepath.Ext(executablePath)); stem != "" {
		values = append(values, normalizedASCII(stem))
	}
	if displayName != "" {
		values = append(values, displayName)
		full, initials := pinyinForms(displayName)
		values = append(values, full, initials)
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.Trim(value, " -")
		if value == "" || Validate(value) != nil {
			continue
		}
		key := Key(value)
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizedASCII(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func pinyinForms(value string) (string, string) {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	parts := pinyin.Pinyin(value, args)
	full := make([]string, 0, len(parts))
	var initials strings.Builder
	for _, group := range parts {
		if len(group) == 0 {
			continue
		}
		part := group[0]
		if part == "" {
			continue
		}
		full = append(full, part)
		for _, r := range part {
			initials.WriteRune(r)
			break
		}
	}
	return normalizedASCII(strings.Join(full, "-")), normalizedASCII(initials.String())
}
