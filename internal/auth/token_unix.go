//go:build !windows

package auth

import (
	"fmt"
	"os"
)

func checkCredentialPermissions(path string, info os.FileInfo) error {
	perm := info.Mode().Perm()
	if perm != 0600 {
		return fmt.Errorf("credentials file has insecure permissions %o (expected 0600): %s", perm, path)
	}
	return nil
}
