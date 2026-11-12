package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// keychainPrefix marks a config value to be read from the macOS Keychain,
// as "keychain:SERVICE/ACCOUNT". This lets WTF share generic passwords that
// other tools already store, e.g.
//
//	security add-generic-password -s glass -a todoist -w
//	apiKey: "keychain:glass/todoist"
const keychainPrefix = "keychain:"

// ResolveSecret returns value, or the Keychain password it refers to.
func ResolveSecret(value string) (string, error) {
	if !strings.HasPrefix(value, keychainPrefix) {
		return value, nil
	}
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("%q: keychain secrets only work on macOS", value)
	}

	service, account, ok := strings.Cut(strings.TrimPrefix(value, keychainPrefix), "/")
	if !ok || service == "" || account == "" {
		return "", fmt.Errorf("%q: use keychain:SERVICE/ACCOUNT", value)
	}

	out, err := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", fmt.Errorf("no keychain item for service %q, account %q", service, account)
	}
	return strings.TrimSpace(string(out)), nil
}
