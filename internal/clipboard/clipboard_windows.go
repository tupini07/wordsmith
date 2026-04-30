//go:build windows
// +build windows

package clipboard

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	getClipboardData = user32.NewProc("GetClipboardData")
	setClipboardData = user32.NewProc("SetClipboardData")
	globalAlloc      = kernel32.NewProc("GlobalAlloc")
	globalFree       = kernel32.NewProc("GlobalFree")
	globalLock       = kernel32.NewProc("GlobalLock")
	globalUnlock     = kernel32.NewProc("GlobalUnlock")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// platformWrite copies text to the Windows clipboard using the Win32 API.
func platformWrite(text string) error {
	r, _, err := openClipboard.Call(0)
	if r == 0 {
		return fmt.Errorf("OpenClipboard: %w", err)
	}
	defer closeClipboard.Call()

	emptyClipboard.Call()

	// Strip any NUL bytes (invalid in clipboard text, causes StringToUTF16 panic)
	text = strings.ReplaceAll(text, "\x00", "")

	// Convert to UTF-16 with null terminator
	utf16 := syscall.StringToUTF16(text)
	size := len(utf16) * 2 // UTF-16 = 2 bytes per code unit

	h, _, err := globalAlloc.Call(gmemMoveable, uintptr(size))
	if h == 0 {
		return fmt.Errorf("GlobalAlloc: %w", err)
	}

	ptr, _, err := globalLock.Call(h)
	if ptr == 0 {
		globalFree.Call(h)
		return fmt.Errorf("GlobalLock: %w", err)
	}

	// Copy UTF-16 data into the allocated memory
	src := unsafe.Pointer(&utf16[0])
	dst := unsafe.Pointer(ptr)
	copy(
		unsafe.Slice((*byte)(dst), size),
		unsafe.Slice((*byte)(src), size),
	)

	globalUnlock.Call(h)

	r, _, err = setClipboardData.Call(cfUnicodeText, h)
	if r == 0 {
		globalFree.Call(h)
		return fmt.Errorf("SetClipboardData: %w", err)
	}
	// After SetClipboardData succeeds, the system owns the memory — don't free it.
	return nil
}

// platformRead reads text from the Windows clipboard using the Win32 API.
func platformRead() (string, error) {
	r, _, err := openClipboard.Call(0)
	if r == 0 {
		return "", fmt.Errorf("OpenClipboard: %w", err)
	}
	defer closeClipboard.Call()

	h, _, _ := getClipboardData.Call(cfUnicodeText)
	if h == 0 {
		// No text on clipboard
		return "", nil
	}

	ptr, _, err := globalLock.Call(h)
	if ptr == 0 {
		return "", fmt.Errorf("GlobalLock: %w", err)
	}
	defer globalUnlock.Call(h)

	// Read UTF-16 string from memory
	text := utf16PtrToString((*uint16)(unsafe.Pointer(ptr)))
	return text, nil
}

// utf16PtrToString converts a null-terminated UTF-16 pointer to a Go string.
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	// Find null terminator
	end := unsafe.Pointer(p)
	n := 0
	for *(*uint16)(unsafe.Pointer(uintptr(end) + uintptr(n)*2)) != 0 {
		n++
	}
	s := unsafe.Slice(p, n)
	return string(syscall.UTF16ToString(s))
}
