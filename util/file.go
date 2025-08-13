package util

import (
	"fmt"
	"mime"
	"path/filepath"
)

func GetFileExtension(filename string) (extension string, err error) {
	// Get the file extension
	ext := filepath.Ext(filename)
	if ext == "" {
		err = fmt.Errorf("file has no extension")
		return
	}

	// Remove the dot
	extension = ext[1:]

	// Get the mime type
	mimeTypes := mime.TypeByExtension("." + extension)

	// If no MIME type found, check if it's a known extension that might not have MIME types
	if mimeTypes == "" {
		// Common file extensions that might not have MIME types on all systems
		knownExtensions := map[string]bool{
			"go":         true, // Go source files
			"mod":        true, // Go module files
			"sum":        true, // Go checksum files
			"md":         true, // Markdown files
			"yml":        true, // YAML files
			"yaml":       true, // YAML files
			"toml":       true, // TOML files
			"gitignore":  true, // Git ignore files
			"dockerfile": true, // Dockerfile
			"makefile":   true, // Makefile
			"sh":         true, // Shell scripts
			"bat":        true, // Batch files
			"ps1":        true, // PowerShell scripts
		}

		if !knownExtensions[extension] {
			err = fmt.Errorf("unknown file extension: %s", extension)
			return
		}
	}

	return
}

func GetDirectoryFile(path string) (directory string, err error) {
	// Get the directory part of the file path
	directory = filepath.Dir(filepath.Clean(path))
	if directory == "." {
		directory = ""
	}

	// Normalize to forward slashes for consistent cross-platform behavior
	directory = filepath.ToSlash(directory)

	// Check if the directory exists
	if _, err = filepath.Abs(filepath.FromSlash(directory)); err != nil {
		return
	}

	return
}

func GetFileName(path string) (filename string, err error) {
	// Get the base name of the file
	filename = filepath.Base(path)
	if filename == "" {
		err = fmt.Errorf("file name is empty")
		return
	}

	// Check if the file exists
	if _, err = filepath.Abs(path); err != nil {
		return
	}

	return
}
