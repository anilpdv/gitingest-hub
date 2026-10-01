package ingest

import (
	"path/filepath"
	"strings"
)

var defaultIgnoredDirs = map[string]bool{
	".git":          true,
	".github":       false, // allow workflows/configs if needed
	".svn":          true,
	".hg":           true,
	".idea":         true,
	".vscode":       true,
	"node_modules":  true,
	"vendor":        true,
	"venv":          true,
	".venv":         true,
	"env":           true,
	".env":          true,
	"__pycache__":   true,
	".pytest_cache": true,
	".mypy_cache":   true,
	"dist":          true,
	"build":         true,
	"out":           true,
	"target":        true,
	"bin":           true,
	"obj":           true,
	"coverage":      true,
	".next":         true,
	".nuxt":         true,
	"xcuserdata":    true,
}

var defaultIgnoredFiles = map[string]bool{
	".DS_Store":         true,
	"Thumbs.db":         true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"composer.lock":     true,
	"Cargo.lock":        true,
	"Gemfile.lock":      true,
	"poetry.lock":       true,
	"go.sum":            false,
	"project.pbxproj":   true, // Xcode project graph (not source code)
}

var binaryExtensions = map[string]bool{
	".exe":   true,
	".dll":   true,
	".so":    true,
	".dylib": true,
	".a":     true,
	".lib":   true,
	".zip":   true,
	".tar":   true,
	".gz":    true,
	".7z":    true,
	".rar":   true,
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".ico":   true,
	".svg":   false, // text-based SVG can be included or skipped
	".webp":  true,
	".pdf":   true,
	".mp4":   true,
	".mp3":   true,
	".wav":   true,
	".mov":   true,
	".avi":   true,
	".woff":  true,
	".woff2": true,
	".eot":   true,
	".ttf":   true,
	".otf":   true,
	".gpx":   true, // GPS spatial trace XML (often 5-10MB each)
	".kml":   true,
	".kmz":   true,
	".fit":   true,
	".tcx":   true,
	".geojson": true,
	".parquet": true,
	".sqlite":  true,
	".sqlite3": true,
	".db":      true,
	".pbxproj":          true,
	".xcworkspacedata":  true,
	".xcscheme":         true,
	".xcuserstate":      true,
	".xccheckout":       true,
}

// ShouldIgnoreFile returns true if the file path should be excluded from ingestion.
func ShouldIgnoreFile(relPath string) bool {
	cleanPath := filepath.ToSlash(relPath)
	parts := strings.Split(cleanPath, "/")

	for _, part := range parts {
		if defaultIgnoredDirs[part] {
			return true
		}
		if defaultIgnoredFiles[part] {
			return true
		}
		if strings.HasSuffix(part, ".xcodeproj") || strings.HasSuffix(part, ".xcworkspace") {
			return true
		}
	}

	baseName := filepath.Base(cleanPath)
	if defaultIgnoredFiles[baseName] {
		return true
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	if binaryExtensions[ext] {
		return true
	}

	// Filter out minified files
	if strings.HasSuffix(cleanPath, ".min.js") || strings.HasSuffix(cleanPath, ".min.css") {
		return true
	}

	return false
}

// IsBinaryContent checks if bytes contain null characters or high concentration of non-printable bytes.
func IsBinaryContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	checkLen := len(data)
	if checkLen > 1024 {
		checkLen = 1024
	}
	for _, b := range data[:checkLen] {
		if b == 0 {
			return true
		}
	}
	return false
}
