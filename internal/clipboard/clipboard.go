package clipboard

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Write copies text to the system clipboard.
func Write(text string) error {
	cmd := writeCmd()
	if cmd == nil {
		return nil
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Read returns the current contents of the system clipboard.
func Read() (string, error) {
	cmd := readCmd()
	if cmd == nil {
		return "", nil
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// powershell.exe appends \r\n; normalize
	s := strings.TrimRight(string(out), "\r\n")
	return s, nil
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
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
	case "windows":
		return exec.Command("clip")
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
