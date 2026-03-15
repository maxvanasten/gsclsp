package analysis

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestURIToPath(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantUnix string
		wantWin  string
	}{
		{
			name:     "empty URI",
			uri:      "",
			wantUnix: "",
			wantWin:  "",
		},
		{
			name:     "simple Unix file URI",
			uri:      "file:///home/user/project/test.gsc",
			wantUnix: "/home/user/project/test.gsc",
			wantWin:  "/home/user/project/test.gsc",
		},
		{
			name:     "Unix URI without file:// prefix",
			uri:      "/home/user/project/test.gsc",
			wantUnix: "/home/user/project/test.gsc",
			wantWin:  "/home/user/project/test.gsc",
		},
		{
			name:     "Windows file URI with drive letter",
			uri:      "file:///C:/Users/project/test.gsc",
			wantUnix: "C:/Users/project/test.gsc", // Leading / stripped for drive letter
			wantWin:  "C:\\Users\\project\\test.gsc",
		},
		{
			name:     "Windows file URI with lowercase drive",
			uri:      "file:///d:/projects/test.gsc",
			wantUnix: "d:/projects/test.gsc", // Leading / stripped for drive letter
			wantWin:  "d:\\projects\\test.gsc",
		},
		{
			name:     "Windows file URI with backslashes",
			uri:      "file:///C:/Users/project/test.gsc",
			wantUnix: "C:/Users/project/test.gsc", // Leading / stripped for drive letter
			wantWin:  "C:\\Users\\project\\test.gsc",
		},
		{
			name:     "relative path without file://",
			uri:      "project/test.gsc",
			wantUnix: "project/test.gsc",
			wantWin:  "project/test.gsc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uriToPath(tt.uri)
			var want string
			if runtime.GOOS == "windows" {
				want = tt.wantWin
			} else {
				want = tt.wantUnix
			}

			// Normalize paths for comparison
			got = filepath.Clean(got)
			want = filepath.Clean(want)

			if got != want {
				t.Errorf("uriToPath(%q) = %q, want %q", tt.uri, got, want)
			}
		})
	}
}

func TestPathToURI(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantUnix string
		wantWin  string
	}{
		{
			name:     "empty path",
			path:     "",
			wantUnix: "",
			wantWin:  "",
		},
		{
			name:     "Unix absolute path",
			path:     "/home/user/project/test.gsc",
			wantUnix: "file:///home/user/project/test.gsc",
			wantWin:  "file:///home/user/project/test.gsc",
		},
		{
			name:     "Unix relative path",
			path:     "project/test.gsc",
			wantUnix: "file:///", // Will have absolute path appended
			wantWin:  "file:///", // Will have absolute path appended
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathToURI(tt.path)
			var want string
			if runtime.GOOS == "windows" {
				want = tt.wantWin
			} else {
				want = tt.wantUnix
			}

			if tt.path == "" {
				if got != "" {
					t.Errorf("pathToURI(%q) = %q, want %q", tt.path, got, want)
				}
				return
			}

			// For relative paths, just check it starts with file://
			if !filepath.IsAbs(tt.path) {
				if !isValidFileURI(got) {
					t.Errorf("pathToURI(%q) = %q, expected valid file URI", tt.path, got)
				}
				return
			}

			if got != want {
				t.Errorf("pathToURI(%q) = %q, want %q", tt.path, got, want)
			}
		})
	}
}

func isValidFileURI(uri string) bool {
	return len(uri) > 7 && uri[:7] == "file://"
}

// Test round-trip conversion for both platforms
func TestURIRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		tests := []struct {
			name string
			uri  string
		}{
			{
				name: "Windows drive letter URI",
				uri:  "file:///C:/Users/project/test.gsc",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				path := uriToPath(tt.uri)
				backToURI := pathToURI(path)

				// The URI should be valid
				if !isValidFileURI(backToURI) {
					t.Errorf("Round-trip produced invalid URI: %q", backToURI)
				}
			})
		}
	} else {
		tests := []struct {
			name string
			uri  string
		}{
			{
				name: "Unix absolute path URI",
				uri:  "file:///home/user/project/test.gsc",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				path := uriToPath(tt.uri)
				backToURI := pathToURI(path)

				if backToURI != tt.uri {
					t.Errorf("Round-trip failed: %q -> %q -> %q, expected %q",
						tt.uri, path, backToURI, tt.uri)
				}
			})
		}
	}
}
