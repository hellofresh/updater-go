//go:build !windows
// +build !windows

package updater

func hideFile(path string) error {
	return nil
}
