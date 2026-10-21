package next

import "strings"

// SelectUTF8Locale returns a UTF-8 locale name exactly as reported by the
// remote host. C.UTF-8 is preferred because it is language-neutral and is
// commonly available even on minimal Linux installations.
func SelectUTF8Locale(available string) string {
	var locales []string
	for _, line := range strings.Split(available, "\n") {
		if locale := strings.TrimSpace(line); locale != "" {
			locales = append(locales, locale)
		}
	}

	preferred := []string{
		"c.utf-8",
		"c.utf8",
		"en_us.utf-8",
		"en_us.utf8",
		"zh_cn.utf-8",
		"zh_cn.utf8",
	}
	for _, want := range preferred {
		for _, locale := range locales {
			if strings.EqualFold(locale, want) {
				return locale
			}
		}
	}
	for _, locale := range locales {
		normalized := strings.ToLower(locale)
		if strings.Contains(normalized, "utf-8") || strings.Contains(normalized, "utf8") {
			return locale
		}
	}
	return ""
}
