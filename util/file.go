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
	if mimeTypes == "" {
		err = fmt.Errorf("unknown file extension: %s", extension)
		return
	}

	return
}

func GetDirectoryFile(path string) (directory string, err error) {
	// Get the directory part of the file path
	directory = filepath.Dir(filepath.Clean(filepath.ToSlash(path)))
	if directory == "." {
		directory = ""
	}

	// Check if the directory exists
	if _, err = filepath.Abs(directory); err != nil {
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
