package chunker

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractGoASTSignatures parses Go source code and extracts types, interfaces, structs,
// and function signatures while omitting function implementation bodies.
func ExtractGoASTSignatures(content string) string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		// Fallback to text compaction if unparseable
		return CompressContentForAI(content)
	}

	// Filter and elide function bodies
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Body != nil {
				// Replace body with a minimal comment placeholder
				fn.Body = &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.BasicLit{
								Kind:  token.STRING,
								Value: "\"// ... implementation elided ...\"",
							},
						},
					},
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return CompressContentForAI(content)
	}

	out := buf.String()
	out = strings.ReplaceAll(out, "\"// ... implementation elided ...\"", "// ... implementation elided ...")
	return CompressContentForAI(out)
}

var (
	// Regex patterns for multi-language AST signature extraction
	pyDefPattern       = regexp.MustCompile(`(?m)^([ \t]*)(async\s+)?def\s+([a-zA-Z0-9_]+)\s*\((.*?)\)(\s*->\s*[^:]+)?:\s*`)
	pyClassPattern     = regexp.MustCompile(`(?m)^class\s+[a-zA-Z0-9_]+(\(.*?\))?:`)
	tsFunctionPattern  = regexp.MustCompile(`^\s*(export\s+)?(default\s+)?(async\s+)?function\s*(\*?\s*)([a-zA-Z0-9_$]+)\s*(<.*?>)?\s*\((.*?)\)(\s*:\s*[^{]+)?\s*\{`)
	tsMethodPattern    = regexp.MustCompile(`^\s*(public\s+|private\s+|protected\s+|async\s+|static\s+)+([a-zA-Z0-9_$]+)\s*(<.*?>)?\s*\((.*?)\)(\s*:\s*[^{]+)?\s*\{`)
	rustFnPattern      = regexp.MustCompile(`^\s*(pub(\(crate\))?\s+)?(async\s+)?fn\s+([a-zA-Z0-9_]+)\s*(<.*?>)?\s*\((.*?)\)(\s*->\s*[^{]+)?\s*\{`)
	swiftFuncPattern   = regexp.MustCompile(`^\s*(public\s+|private\s+|fileprivate\s+|internal\s+|open\s+|static\s+)*(func)\s+([a-zA-Z0-9_]+)\s*(<.*?>)?\s*\((.*?)\)(\s*(async|throws|\s)*->\s*[^{]+)?\s*\{`)
	javaMethodPattern  = regexp.MustCompile(`^\s*(public\s+|protected\s+|private\s+|static\s+|final\s+|native\s+|synchronized\s+)+([a-zA-Z0-9_<>[\]]+)\s+([a-zA-Z0-9_]+)\s*\((.*?)\)(\s*throws\s+[^{]+)?\s*\{`)
)

// ExtractPythonSignatures extracts classes, methods, signatures, and docstrings from Python source code.
func ExtractPythonSignatures(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inDocstring := false
	docstringMarker := ""

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Preserve docstrings for context
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			marker := trimmed[:3]
			if !inDocstring {
				inDocstring = true
				docstringMarker = marker
				result = append(result, line)
				if len(trimmed) > 3 && strings.HasSuffix(trimmed[3:], marker) {
					inDocstring = false
				}
				continue
			} else if marker == docstringMarker {
				inDocstring = false
				result = append(result, line)
				continue
			}
		}
		if inDocstring {
			result = append(result, line)
			continue
		}

		// Keep imports, decorators, class headers, type aliases, and def headers
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") ||
			strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") ||
			strings.Contains(trimmed, " = TypeVar(") || strings.Contains(trimmed, " = NewType(") ||
			strings.HasPrefix(trimmed, "class ") {
			
			result = append(result, line)

			// If it's a function or method header, append an elided body stub
			if (strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ")) && strings.HasSuffix(trimmed, ":") {
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				result = append(result, indent+"    ... # implementation elided")
			}
		} else if strings.HasPrefix(trimmed, "#") {
			// Keep top comments
			result = append(result, line)
		}
	}

	return CompressContentForAI(strings.Join(result, "\n"))
}

// ExtractTypeScriptSignatures extracts interfaces, types, exported signatures, and classes from TS/JS code.
func ExtractTypeScriptSignatures(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Preserve import / export type statements, interfaces, enums, type definitions
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export type ") ||
			strings.HasPrefix(trimmed, "export interface ") || strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "type ") || strings.HasPrefix(trimmed, "enum ") ||
			strings.HasPrefix(trimmed, "export enum ") || strings.HasPrefix(trimmed, "export class ") ||
			strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "export const ") ||
			strings.HasPrefix(trimmed, "declare ") {
			result = append(result, line)
		} else if tsFunctionPattern.MatchString(line) {
			// Replace { with { /* ... implementation elided ... */ }
			header := tsFunctionPattern.ReplaceAllString(line, "${1}${2}${3}function ${4}${5}${6}(${7})${8} { /* ... elided ... */ }")
			result = append(result, header)
		} else if tsMethodPattern.MatchString(line) && !strings.Contains(trimmed, "if ") && !strings.Contains(trimmed, "for ") && !strings.Contains(trimmed, "switch ") {
			header := tsMethodPattern.ReplaceAllString(line, "${1}${2}${3}(${4})${5} { /* ... elided ... */ }")
			result = append(result, header)
		}
	}

	if len(result) == 0 {
		return CompressContentForAI(content)
	}

	return CompressContentForAI(strings.Join(result, "\n"))
}

// ExtractASTSignatures routes to the optimal AST / grammar-aware extractor based on file extension.
func ExtractASTSignatures(path string, content string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".go":
		return ExtractGoASTSignatures(content)
	case ".py", ".pyw":
		return ExtractPythonSignatures(content)
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		return ExtractTypeScriptSignatures(content)
	case ".rs":
		// Fast signature extractor for Rust
		lines := strings.Split(content, "\n")
		var result []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "use ") || strings.HasPrefix(trimmed, "pub struct ") ||
				strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "pub enum ") ||
				strings.HasPrefix(trimmed, "enum ") || strings.HasPrefix(trimmed, "pub trait ") ||
				strings.HasPrefix(trimmed, "trait ") || strings.HasPrefix(trimmed, "impl ") {
				result = append(result, line)
			} else if rustFnPattern.MatchString(line) {
				sig := rustFnPattern.ReplaceAllString(line, "$1$2$4fn $5$6($7)$8 { /* ... elided ... */ }")
				result = append(result, sig)
			}
		}
		if len(result) > 0 {
			return CompressContentForAI(strings.Join(result, "\n"))
		}
		return CompressContentForAI(content)
	case ".swift":
		// Fast signature extractor for Swift
		lines := strings.Split(content, "\n")
		var result []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "protocol ") ||
				strings.HasPrefix(trimmed, "public protocol ") || strings.HasPrefix(trimmed, "struct ") ||
				strings.HasPrefix(trimmed, "public struct ") || strings.HasPrefix(trimmed, "class ") ||
				strings.HasPrefix(trimmed, "public class ") || strings.HasPrefix(trimmed, "enum ") ||
				strings.HasPrefix(trimmed, "extension ") {
				result = append(result, line)
			} else if swiftFuncPattern.MatchString(line) {
				sig := swiftFuncPattern.ReplaceAllString(line, "$1$2$3 $4$5($6)$7 { /* ... elided ... */ }")
				result = append(result, sig)
			}
		}
		if len(result) > 0 {
			return CompressContentForAI(strings.Join(result, "\n"))
		}
		return CompressContentForAI(content)
	default:
		// For other files, apply clean token & boilerplate compaction
		return CompressContentForAI(content)
	}
}
