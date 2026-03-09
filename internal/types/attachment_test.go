package types

import "testing"

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
		if got := a.IsImage(); got != tt.want {
			t.Errorf("IsImage(%q) = %v, want %v", tt.contentType, got, tt.want)
		}
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
		if got := a.HumanSize(); got != tt.want {
			t.Errorf("HumanSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}
