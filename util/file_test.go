package util

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		expectedExt   string
		expectedError bool
		errorContains string
	}{
		{
			name:        "Valid image file - jpg",
			filename:    "photo.jpg",
			expectedExt: "jpg",
		},
		{
			name:        "Valid image file - png",
			filename:    "image.png",
			expectedExt: "png",
		},
		{
			name:        "Valid document - pdf",
			filename:    "document.pdf",
			expectedExt: "pdf",
		},
		{
			name:        "Valid text file",
			filename:    "readme.txt",
			expectedExt: "txt",
		},
		{
			name:        "Valid code file - go",
			filename:    "main.go",
			expectedExt: "go",
		},
		{
			name:        "Valid web file - html",
			filename:    "index.html",
			expectedExt: "html",
		},
		{
			name:        "Valid data file - json",
			filename:    "config.json",
			expectedExt: "json",
		},
		{
			name:        "Valid data file - xml",
			filename:    "data.xml",
			expectedExt: "xml",
		},
		{
			name:        "Valid compressed file - zip",
			filename:    "archive.zip",
			expectedExt: "zip",
		},
		{
			name:        "File with path",
			filename:    "/path/to/file.css",
			expectedExt: "css",
		},
		{
			name:        "File with Windows path",
			filename:    "C:\\Users\\test\\file.js",
			expectedExt: "js",
		},
		{
			name:        "File with multiple dots",
			filename:    "file.test.json",
			expectedExt: "json",
		},
		{
			name:          "Hidden file with extension",
			filename:      ".gitignore",
			expectedExt:   "gitignore", // Function returns the extension as it's a known type
			expectedError: false,
			errorContains: "",
		},
		{
			name:          "File without extension",
			filename:      "README",
			expectedError: true,
			errorContains: "file has no extension",
		},
		{
			name:          "Empty filename",
			filename:      "",
			expectedError: true,
			errorContains: "file has no extension",
		},
		{
			name:          "Directory without extension",
			filename:      "/path/to/directory",
			expectedError: true,
			errorContains: "file has no extension",
		},
		{
			name:          "File with unknown extension",
			filename:      "file.unknownext123",
			expectedExt:   "unknownext123", // Function still returns the extension even with error
			expectedError: true,
			errorContains: "unknown file extension: unknownext123",
		},
		{
			name:          "Dot only",
			filename:      ".",
			expectedExt:   "", // Empty extension returned
			expectedError: true,
			errorContains: "unknown file extension: ",
		},
		{
			name:          "Double dot",
			filename:      "..",
			expectedExt:   "", // Empty extension returned
			expectedError: true,
			errorContains: "unknown file extension: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extension, err := GetFileExtension(tt.filename)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				if tt.expectedExt != "" {
					assert.Equal(t, tt.expectedExt, extension)
				} else {
					assert.Empty(t, extension)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedExt, extension)
			}
		})
	}
}

func TestGetDirectoryFile(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		expectedDirectory string
		expectedError     bool
		errorContains     string
	}{
		{
			name:              "Simple file path",
			path:              "file.txt",
			expectedDirectory: "",
		},
		{
			name:              "File in directory",
			path:              "documents/file.txt",
			expectedDirectory: "documents",
		},
		{
			name:              "Nested directory path",
			path:              "home/user/documents/file.txt",
			expectedDirectory: "home/user/documents",
		},
		{
			name:              "Absolute Unix path",
			path:              "/home/user/documents/file.txt",
			expectedDirectory: "/home/user/documents",
		},
		{
			name:              "Root directory file",
			path:              "/file.txt",
			expectedDirectory: "/",
		},
		{
			name:              "Current directory reference",
			path:              "./file.txt",
			expectedDirectory: "",
		},
		{
			name:              "Parent directory reference",
			path:              "../file.txt",
			expectedDirectory: "..",
		},
		{
			name:              "Complex path with parent references",
			path:              "dir1/../dir2/file.txt",
			expectedDirectory: "dir2",
		},
		{
			name:              "Directory with trailing slash",
			path:              "documents/",
			expectedDirectory: "",
		},
		{
			name:              "Empty path",
			path:              "",
			expectedDirectory: "",
		},
		{
			name:              "Just directory name",
			path:              "documents",
			expectedDirectory: "",
		},
		{
			name:              "Path with spaces",
			path:              "My Documents/My File.txt",
			expectedDirectory: "My Documents",
		},
		{
			name:              "Path with special characters",
			path:              "documents-2024/file_name.txt",
			expectedDirectory: "documents-2024",
		},
	}

	// Add Windows-specific tests only on Windows
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			name              string
			path              string
			expectedDirectory string
		}{
			{
				name:              "Windows absolute path",
				path:              "C:\\Users\\test\\file.txt",
				expectedDirectory: "C:/Users/test",
			},
			{
				name:              "Windows path with forward slashes",
				path:              "C:/Users/test/file.txt",
				expectedDirectory: "C:/Users/test",
			},
		}
		for _, wt := range windowsTests {
			tests = append(tests, struct {
				name              string
				path              string
				expectedDirectory string
				expectedError     bool
				errorContains     string
			}{
				name:              wt.name,
				path:              wt.path,
				expectedDirectory: wt.expectedDirectory,
			})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory, err := GetDirectoryFile(tt.path)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Empty(t, directory)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDirectory, directory)
			}
		})
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		expectedFilename string
		expectedError    bool
		errorContains    string
	}{
		{
			name:             "Simple filename",
			path:             "file.txt",
			expectedFilename: "file.txt",
		},
		{
			name:             "File in directory",
			path:             "documents/file.txt",
			expectedFilename: "file.txt",
		},
		{
			name:             "Nested directory path",
			path:             "home/user/documents/file.txt",
			expectedFilename: "file.txt",
		},
		{
			name:             "Absolute Unix path",
			path:             "/home/user/documents/file.txt",
			expectedFilename: "file.txt",
		},
		{
			name:             "Root directory file",
			path:             "/file.txt",
			expectedFilename: "file.txt",
		},
		{
			name:             "Current directory reference",
			path:             "./file.txt",
			expectedFilename: "file.txt",
		},
		{
			name:             "Parent directory reference",
			path:             "../file.txt",
			expectedFilename: "file.txt",
		},
		{
			name:             "File without extension",
			path:             "documents/README",
			expectedFilename: "README",
		},
		{
			name:             "Hidden file",
			path:             ".gitignore",
			expectedFilename: ".gitignore",
		},
		{
			name:             "File with spaces",
			path:             "My Documents/My File.txt",
			expectedFilename: "My File.txt",
		},
		{
			name:             "File with special characters",
			path:             "documents/file_name-2024.txt",
			expectedFilename: "file_name-2024.txt",
		},
		{
			name:             "File with multiple dots",
			path:             "config/app.config.json",
			expectedFilename: "app.config.json",
		},
		{
			name:             "Directory name only",
			path:             "documents",
			expectedFilename: "documents",
		},
		{
			name:             "Complex path with parent references",
			path:             "dir1/../dir2/file.txt",
			expectedFilename: "file.txt",
		},
	}

	// Add Windows-specific tests only on Windows
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			name             string
			path             string
			expectedFilename string
		}{
			{
				name:             "Windows absolute path",
				path:             "C:\\Users\\test\\file.txt",
				expectedFilename: "file.txt",
			},
			{
				name:             "Windows path with forward slashes",
				path:             "C:/Users/test/file.txt",
				expectedFilename: "file.txt",
			},
		}
		for _, wt := range windowsTests {
			tests = append(tests, struct {
				name             string
				path             string
				expectedFilename string
				expectedError    bool
				errorContains    string
			}{
				name:             wt.name,
				path:             wt.path,
				expectedFilename: wt.expectedFilename,
			})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename, err := GetFileName(tt.path)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Empty(t, filename)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedFilename, filename)
			}
		})
	}
}

// Benchmark tests to measure performance
func BenchmarkGetFileExtension(b *testing.B) {
	testFiles := []string{
		"file.txt",
		"image.png",
		"document.pdf",
		"code.go",
		"data.json",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range testFiles {
			_, _ = GetFileExtension(file)
		}
	}
}

func BenchmarkGetDirectoryFile(b *testing.B) {
	testPaths := []string{
		"file.txt",
		"documents/file.txt",
		"home/user/documents/file.txt",
		"/home/user/documents/file.txt",
		"../parent/file.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_, _ = GetDirectoryFile(path)
		}
	}
}

func BenchmarkGetFileName(b *testing.B) {
	testPaths := []string{
		"file.txt",
		"documents/file.txt",
		"home/user/documents/file.txt",
		"/home/user/documents/file.txt",
		"../parent/file.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_, _ = GetFileName(path)
		}
	}
}

// Example tests for documentation
func ExampleGetFileExtension() {
	ext, err := GetFileExtension("document.pdf")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Extension: %s\n", ext)
	// Output: Extension: pdf
}

func ExampleGetDirectoryFile() {
	dir, err := GetDirectoryFile("home/user/documents/file.txt")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Directory: %s\n", dir)
	// Output: Directory: home/user/documents
}

func ExampleGetFileName() {
	filename, err := GetFileName("home/user/documents/file.txt")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Filename: %s\n", filename)
	// Output: Filename: file.txt
}

// Integration tests that test multiple functions together
func TestFileUtilsIntegration(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		expectDir   string
		expectFile  string
		expectExt   string
		expectError bool
	}{
		{
			name:       "Complete file path analysis",
			path:       "documents/images/photo.jpg",
			expectDir:  "documents/images",
			expectFile: "photo.jpg",
			expectExt:  "jpg",
		},
		{
			name:       "Root file analysis",
			path:       "/config.json",
			expectDir:  "/",
			expectFile: "config.json",
			expectExt:  "json",
		},
		{
			name:       "Current directory file",
			path:       "README.md",
			expectDir:  "",
			expectFile: "README.md",
			expectExt:  "md",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test directory extraction
			dir, err := GetDirectoryFile(tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.expectDir, dir)

			// Test filename extraction
			filename, err := GetFileName(tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.expectFile, filename)

			// Test extension extraction
			ext, err := GetFileExtension(filename)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectExt, ext)
			}
		})
	}
}

// Test edge cases and error conditions
func TestFileUtilsEdgeCases(t *testing.T) {
	t.Run("Empty string inputs", func(t *testing.T) {
		// Test GetFileExtension with empty string
		ext, err := GetFileExtension("")
		assert.Error(t, err)
		assert.Empty(t, ext)

		// Test GetDirectoryFile with empty string
		dir, err := GetDirectoryFile("")
		assert.NoError(t, err) // This should not error based on current implementation
		assert.Empty(t, dir)

		// Test GetFileName with empty string
		filename, err := GetFileName("")
		assert.NoError(t, err)         // This should not error based on current implementation
		assert.Equal(t, ".", filename) // filepath.Base("") returns "."
	})

	t.Run("Special path characters", func(t *testing.T) {
		specialPaths := []string{
			"file with spaces.txt",
			"file-with-dashes.txt",
			"file_with_underscores.txt",
			"file.with.multiple.dots.txt",
			"file@with@symbols.txt",
		}

		for _, path := range specialPaths {
			t.Run(fmt.Sprintf("Path: %s", path), func(t *testing.T) {
				// These should all work without errors
				dir, err := GetDirectoryFile(path)
				assert.NoError(t, err)
				assert.Empty(t, dir) // All are in current directory

				filename, err := GetFileName(path)
				assert.NoError(t, err)
				assert.Equal(t, path, filename)

				ext, err := GetFileExtension(path)
				assert.NoError(t, err)
				assert.Equal(t, "txt", ext)
			})
		}
	})
}
