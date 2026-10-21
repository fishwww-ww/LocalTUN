package next

import (
	"fmt"
	"strings"
)

func ProxyEnv(proxyURL string) map[string]string {
	return map[string]string{
		"HTTP_PROXY":  proxyURL,
		"HTTPS_PROXY": proxyURL,
		"ALL_PROXY":   proxyURL,
		"http_proxy":  proxyURL,
		"https_proxy": proxyURL,
		"all_proxy":   proxyURL,
	}
}

func ShellCommand(shell string, proxyURL, locale string) string {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		shell = "sh"
	}
	env := ProxyEnv(proxyURL)
	order := []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"}
	locale = strings.TrimSpace(locale)
	if locale != "" {
		env["LANG"] = locale
		env["LC_CTYPE"] = locale
		env["LC_ALL"] = locale
		order = append(order, "LANG", "LC_CTYPE", "LC_ALL")
	}
	parts := []string{"env"}
	for _, key := range order {
		parts = append(parts, fmt.Sprintf("%s=%s", key, shQuote(env[key])))
	}
	parts = append(parts, shQuote(shell), "-l")
	return strings.Join(parts, " ")
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
