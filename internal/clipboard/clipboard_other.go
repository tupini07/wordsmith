//go:build !windows
// +build !windows

package clipboard

// platformWrite uses shell commands on non-Windows platforms.
func platformWrite(text string) error {
	return shellWrite(text)
}

// platformRead uses shell commands on non-Windows platforms.
func platformRead() (string, error) {
	return shellRead()
}
