package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/johanviberg/zd/internal/types"
)

type ErrorResponse struct {
	SchemaVersion string `json:"schemaVersion"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	ExitCode      int    `json:"exitCode"`
	RetryAfter    int    `json:"retryAfter,omitempty"`
}

func PrintError(w io.Writer, err error, jsonOutput bool) {
	if jsonOutput {
		resp := ErrorResponse{
			SchemaVersion: "1.0",
			Code:          "error",
			Message:       err.Error(),
			ExitCode:      1,
		}
		if appErr, ok := err.(*types.AppError); ok {
			resp.Code = appErr.Code
			resp.ExitCode = appErr.ExitCode
			resp.RetryAfter = appErr.RetryAfter
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	fmt.Fprintf(w, "Error: %s\n", err.Error())
}

func ExitWithError(err error, jsonOutput bool) {
	PrintError(os.Stderr, err, jsonOutput)
	exitCode := 1
	if appErr, ok := err.(*types.AppError); ok {
		exitCode = appErr.ExitCode
	}
	os.Exit(exitCode)
}
