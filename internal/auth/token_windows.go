//go:build windows

package auth

import "os"

// Windows does not support Unix file permissions. NTFS ACLs are managed
// separately; os.FileMode always reports 0666 on Windows, so the Unix
// permission check is skipped.
func checkCredentialPermissions(_ string, _ os.FileInfo) error {
	return nil
}
