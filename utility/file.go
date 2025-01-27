package utility

import (
	"fmt"
	"mime"
	"path/filepath"
)

func GetFileExtension(filename string) (extension string, err error) {
	// Get the file extension
	ext := filepath.Ext(filename)
	if ext == "" {
		return "", fmt.Errorf("file extension not found")
	}

	// Remove the dot
	extension = ext[1:]

	// Get the mime type
	mimeTypes := mime.TypeByExtension("." + extension)
	if mimeTypes == "" {
		return "", fmt.Errorf("file extension not found")
	}

	return extension, nil
}
