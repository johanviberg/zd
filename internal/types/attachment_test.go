package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttachment_IsImage(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"image/webp", true},
		{"text/plain", false},
		{"application/pdf", false},
		{"", false},
	}
	for _, tt := range tests {
		a := Attachment{ContentType: tt.contentType}
		assert.Equal(t, tt.want, a.IsImage(), "IsImage(%q)", tt.contentType)
	}
}

func TestAttachment_HumanSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{45200, "44 KB"},
		{1048576, "1.0 MB"},
		{2621440, "2.5 MB"},
	}
	for _, tt := range tests {
		a := Attachment{Size: tt.size}
		assert.Equal(t, tt.want, a.HumanSize(), "HumanSize(%d)", tt.size)
	}
}
