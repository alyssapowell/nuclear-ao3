package works

import (
	"fmt"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The upload handler is not routed yet; its file check is still plain logic.
func TestValidateUploadedFile(t *testing.T) {
	ws := &WorkService{}

	testCases := []struct {
		filename    string
		size        int64
		expectError bool
		errorMsg    string
	}{
		{"test.txt", 1024, false, ""},
		{"test.pdf", 1024, false, ""},
		{"test.doc", 1024, false, ""},
		{"test.docx", 1024, false, ""},
		{"test.epub", 1024, false, ""},
		{"test.html", 1024, false, ""},
		{"test.exe", 1024, true, "file type not allowed"},
		{"test.txt", 11 * 1024 * 1024, true, "file too large (max 10MB)"},
		{"test.jpg", 1024, true, "file type not allowed"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ValidateFile_%s", tc.filename), func(t *testing.T) {
			err := ws.validateUploadedFile(&multipart.FileHeader{Filename: tc.filename, Size: tc.size})
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
