package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintError_JSON_AppError(t *testing.T) {
	var buf bytes.Buffer
	err := types.NewRetryableError("too fast", 30)
	PrintError(&buf, err, true)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &resp))
	assert.Equal(t, "rate_limited", resp["code"])
	assert.Equal(t, float64(4), resp["exitCode"])
	assert.Equal(t, float64(30), resp["retryAfter"])
}

func TestPrintError_PlainText(t *testing.T) {
	var buf bytes.Buffer
	PrintError(&buf, fmt.Errorf("something broke"), false)
	assert.Contains(t, buf.String(), "Error: something broke")
}
