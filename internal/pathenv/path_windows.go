//go:build windows

package pathenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const environmentKey = `Environment`

func Contains(dir string) bool {
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if samePath(entry, dir) {
			return true
		}
	}
	return false
}

func Ensure(dir string) (bool, error) {
	path, valueType, key, err := readUserPath()
	if err != nil {
		return false, err
	}
	defer key.Close()
	entries := nonEmpty(filepath.SplitList(path))
	for _, entry := range entries {
		if samePath(entry, dir) {
			return false, nil
		}
	}
	entries = append(entries, dir)
	if err := writeUserPath(key, strings.Join(entries, ";"), valueType); err != nil {
		return false, err
	}
	broadcastEnvironmentChange()
	return true, nil
}

func Remove(dir string) (bool, error) {
	path, valueType, key, err := readUserPath()
	if err != nil {
		return false, err
	}
	defer key.Close()
	entries := nonEmpty(filepath.SplitList(path))
	filtered := entries[:0]
	changed := false
	for _, entry := range entries {
		if samePath(entry, dir) {
			changed = true
			continue
		}
		filtered = append(filtered, entry)
	}
	if !changed {
		return false, nil
	}
	if err := writeUserPath(key, strings.Join(filtered, ";"), valueType); err != nil {
		return false, err
	}
	broadcastEnvironmentChange()
	return true, nil
}

func readUserPath() (string, uint32, registry.Key, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, environmentKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return "", 0, 0, fmt.Errorf("open user environment registry key: %w", err)
	}
	value, valueType, err := key.GetStringValue("Path")
	if err == registry.ErrNotExist {
		return "", registry.SZ, key, nil
	}
	if err != nil {
		key.Close()
		return "", 0, 0, fmt.Errorf("read user PATH: %w", err)
	}
	return value, valueType, key, nil
}

func writeUserPath(key registry.Key, value string, valueType uint32) error {
	if valueType == registry.EXPAND_SZ {
		return key.SetExpandStringValue("Path", value)
	}
	return key.SetStringValue("Path", value)
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(a, b)
	}
	return strings.EqualFold(aa, bb)
}

func broadcastEnvironmentChange() {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	name, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	var result uintptr
	const (
		hwndBroadcast    = uintptr(0xffff)
		wmSettingChange  = uintptr(0x001a)
		smtoAbortIfHung  = uintptr(0x0002)
	)
	proc.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(name)), smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}
