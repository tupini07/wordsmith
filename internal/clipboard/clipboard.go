package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrUnavailable is returned when no clipboard backend is available on the
// current system (e.g. Linux without xclip/xsel/wl-clipboard installed).
// Callers can use UnavailableMessage to render an OS-specific install hint.
var ErrUnavailable = errors.New("clipboard backend unavailable")

// UnavailableMessage returns a human-readable, OS-specific hint explaining how
// to make the clipboard work on the current system.
func UnavailableMessage() string {
	switch runtime.GOOS {
	case "linux":
		if isWSL() {
			return "Clipboard unavailable: clip.exe / powershell.exe not found in PATH"
		}
		return "Clipboard unavailable: install xclip, xsel, or wl-clipboard"
	case "darwin":
		return "Clipboard unavailable: pbcopy/pbpaste not found"
	case "windows":
		return "Clipboard unavailable"
	default:
		return "Clipboard unavailable on " + runtime.GOOS
	}
}

// Write copies text to the system clipboard.
func Write(text string) error {
	return platformWrite(text)
}

// Read returns the current contents of the system clipboard.
func Read() (string, error) {
	return platformRead()
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

// shellWrite is the fallback clipboard write using external commands.
func shellWrite(text string) error {
	cmd := writeCmd()
	if cmd == nil {
		return ErrUnavailable
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// shellRead is the fallback clipboard read using external commands.
func shellRead() (string, error) {
	cmd := readCmd()
	if cmd == nil {
		return "", ErrUnavailable
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// powershell.exe appends \r\n; normalize
	s := strings.TrimRight(string(out), "\r\n")
	return s, nil
}

func writeCmd() *exec.Cmd {
	if isWSL() {
		if p, err := exec.LookPath("clip.exe"); err == nil {
			return exec.Command(p)
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("pbcopy")
	case "linux":
		if p, err := exec.LookPath("xclip"); err == nil {
			return exec.Command(p, "-selection", "clipboard")
		}
		if p, err := exec.LookPath("xsel"); err == nil {
			return exec.Command(p, "--clipboard", "--input")
		}
		if p, err := exec.LookPath("wl-copy"); err == nil {
			return exec.Command(p)
		}
	}
	return nil
}

func readCmd() *exec.Cmd {
	if isWSL() {
		if p, err := exec.LookPath("powershell.exe"); err == nil {
			return exec.Command(p, "-NoProfile", "-command", "Get-Clipboard")
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("pbpaste")
	case "linux":
		if p, err := exec.LookPath("xclip"); err == nil {
			return exec.Command(p, "-selection", "clipboard", "-o")
		}
		if p, err := exec.LookPath("xsel"); err == nil {
			return exec.Command(p, "--clipboard", "--output")
		}
		if p, err := exec.LookPath("wl-paste"); err == nil {
			return exec.Command(p)
		}
	}
	return nil
}
