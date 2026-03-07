package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open opens the given URL in the user's default browser.
func Open(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Env = []string{}
		if err := cmd.Start(); err != nil {
			fmt.Printf("Warning: could not open browser: %v\n", err)
		}
	}
}
