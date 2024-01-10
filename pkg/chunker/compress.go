package chunker

import (
	"regexp"
	"strings"

	"gitingest-hub/pkg/model"
)

var (
	// Regex for top-of-file block comment license/copyright
	cStyleLicenseBlock = regexp.MustCompile(`(?s)^\s*/\*.*?(Copyright|License|Licensed|SPDX-License-Identifier).*?\*/\s*`)
	// Regex for top-of-file line comment license/copyright (Go, JS, TS, Rust, Swift, C++, Java, etc.)
	slashSlashLicense = regexp.MustCompile(`(?m)^(//\s*(Copyright|License|Licensed|SPDX-License-Identifier|All rights reserved).*?\n)+`)
	// Regex for top-of-file Python / Shell / Ruby hash comments
	hashLicense = regexp.MustCompile(`(?m)^(#\s*(Copyright|License|Licensed|SPDX-License-Identifier|All rights reserved).*?\n)+`)
	// Regex for 3+ consecutive newlines
	multiNewlines = regexp.MustCompile(`\n{3,}`)
)

// CompressContentForAI trims non-essential token overhead (license headers, trailing spaces, redundant blank lines)
// without modifying any executable code logic, variable names, or indentation structure.
func CompressContentForAI(content string) string {
	if len(content) == 0 {
		return content
	}

	trimmed := content

	// 1. Strip top-of-file boilerplate license blocks
	trimmed = cStyleLicenseBlock.ReplaceAllString(trimmed, "")
	trimmed = slashSlashLicense.ReplaceAllString(trimmed, "")
	trimmed = hashLicense.ReplaceAllString(trimmed, "")

	// 2. Normalize line breaks and trim trailing whitespace per line
	lines := strings.Split(trimmed, "\n")
	var cleanedLines []string
	for _, line := range lines {
		cleanedLines = append(cleanedLines, strings.TrimRight(line, " \t\r"))
	}
	trimmed = strings.Join(cleanedLines, "\n")

	// 3. Collapse 3+ consecutive empty lines down to a single blank line (\n\n)
	trimmed = multiNewlines.ReplaceAllString(trimmed, "\n\n")

	return strings.TrimSpace(trimmed)
}

// ProcessFileContent applies the requested compression tier to a single file.
func ProcessFileContent(file model.IngestFile, mode string) string {
	switch mode {
	case "ast_signatures":
		return ExtractASTSignatures(file.Path, file.Content)
	case "compact":
		return CompressContentForAI(file.Content)
	case "none":
		fallthrough
	default:
		return file.Content
	}
}

// FormatFileBlockWithOption formats a file either in standard GitIngest style or AI-Optimized XML style.
func FormatFileBlockWithOption(file model.IngestFile, compressMode string) string {
	if compressMode == "ast_signatures" || compressMode == "compact" {
		content := ProcessFileContent(file, compressMode)
		var sb strings.Builder
		sb.WriteString("<file path=\"")
		sb.WriteString(file.Path)
		sb.WriteString("\">\n")
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("</file>\n\n")
		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString("================================================\n")
	sb.WriteString("FILE: ")
	sb.WriteString(file.Path)
	sb.WriteString("\n================================================\n")
	sb.WriteString(file.Content)
	if !strings.HasSuffix(file.Content, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}
