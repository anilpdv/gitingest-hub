package ingest

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gitingest-hub/pkg/model"
)

// ParsedRepoURL extracts owner, repo, ref/branch, and subpath from a GitHub URL.
type ParsedRepoURL struct {
	Owner   string
	Repo    string
	Branch  string
	Subpath string
}

// ParseGitHubURL parses strings like https://github.com/owner/repo or owner/repo/tree/branch/path.
func ParseGitHubURL(rawURL string) (*ParsedRepoURL, error) {
	clean := strings.TrimSpace(rawURL)
	clean = strings.TrimPrefix(clean, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "github.com/")
	clean = strings.TrimSuffix(clean, ".git")
	clean = strings.Trim(clean, "/")

	parts := strings.Split(clean, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub repository URL: %s (expected owner/repo format)", rawURL)
	}

	result := &ParsedRepoURL{
		Owner: parts[0],
		Repo:  parts[1],
	}

	if len(parts) >= 4 && (parts[2] == "tree" || parts[2] == "blob") {
		result.Branch = parts[3]
		if len(parts) > 4 {
			result.Subpath = strings.Join(parts[4:], "/")
		}
	} else if len(parts) > 2 {
		// User passed direct subfolder path, e.g. https://github.com/anilpdv/UltraNav/Domain
		result.Subpath = strings.Join(parts[2:], "/")
	}

	result.Subpath = strings.Trim(result.Subpath, "/")

	return result, nil
}

// EstimateTokens calculates an approximate token count for code and text (~3.8 to 4 chars per token).
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	charCount := utf8.RuneCountInString(text)
	// Code has many symbols and indentation, average ratio is ~3.7-4 chars/token
	tokens := int(float64(charCount) / 3.7)
	if tokens == 0 && charCount > 0 {
		return 1
	}
	return tokens
}

// FetchRepositoryStreams streams a repository tarball directly into memory and extracts valid files.
func FetchRepository(opts model.IngestOptions) ([]model.IngestFile, *ParsedRepoURL, error) {
	parsed, err := ParseGitHubURL(opts.RepoURL)
	if err != nil {
		return nil, nil, err
	}

	branch := opts.Branch
	if branch == "" {
		branch = parsed.Branch
	}
	if branch == "" {
		branch = "HEAD"
	}

	// GitHub Tarball URL
	tarballURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s",
		url.PathEscape(parsed.Owner),
		url.PathEscape(parsed.Repo),
		url.PathEscape(branch),
	)

	client := &http.Client{
		Timeout: 45 * time.Second,
	}

	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "GitIngest-Hub-Go/1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch repository tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If HEAD fails, try "main" and "master" as fallbacks
		if branch == "HEAD" && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden) {
			return nil, nil, fmt.Errorf("repository %s/%s not found, is private, or rate limited (HTTP %d)", parsed.Owner, parsed.Repo, resp.StatusCode)
		}
		return nil, nil, fmt.Errorf("GitHub API returned status %d for %s/%s", resp.StatusCode, parsed.Owner, parsed.Repo)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decompress gzip archive: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var files []model.IngestFile

	// GitHub tarballs have a root directory like "owner-repo-commitSHA/"
	var rootPrefix string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error reading tar archive: %w", err)
		}

		if rootPrefix == "" {
			parts := strings.Split(header.Name, "/")
			if len(parts) > 1 {
				rootPrefix = parts[0] + "/"
			}
		}

		relPath := strings.TrimPrefix(header.Name, rootPrefix)
		if relPath == "" || header.Typeflag != tar.TypeReg {
			continue
		}

		// Filter out based on subpath if specified (e.g. "Domain" or "UltraNav/Domain")
		if parsed.Subpath != "" {
			cleanSubpath := strings.Trim(parsed.Subpath, "/")
			// Check exact prefix match (e.g. "Domain/...") or with nested prefix (e.g. "UltraNav/Domain/...")
			matched := strings.HasPrefix(relPath, cleanSubpath+"/") || 
				relPath == cleanSubpath ||
				strings.Contains(relPath, "/"+cleanSubpath+"/") ||
				strings.HasSuffix(relPath, "/"+cleanSubpath) ||
				strings.HasPrefix(strings.ToLower(relPath), strings.ToLower(cleanSubpath)+"/") ||
				strings.Contains(strings.ToLower(relPath), "/"+strings.ToLower(cleanSubpath)+"/")
			
			if !matched {
				continue
			}
		}

		if ShouldIgnoreFile(relPath) {
			continue
		}

		// Ignore individual files larger than 512KB to prevent memory and browser DOM rendering lockup
		if header.Size > 512*1024 {
			continue
		}

		buf := make([]byte, header.Size)
		_, err = io.ReadFull(tarReader, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			continue
		}

		if IsBinaryContent(buf) {
			continue
		}

		contentStr := string(buf)
		files = append(files, model.IngestFile{
			Path:       filepath.ToSlash(relPath),
			Size:       header.Size,
			Content:    contentStr,
			TokenCount: EstimateTokens(contentStr),
		})
	}

	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no readable code/text files found in repository %s/%s", parsed.Owner, parsed.Repo)
	}

	// Sort files alphabetically
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, parsed, nil
}

// GenerateDirectoryTree generates an ASCII tree structure representing the codebase.
func GenerateDirectoryTree(repoName string, files []model.IngestFile) string {
	type TreeNode struct {
		Name     string
		IsDir    bool
		Children map[string]*TreeNode
	}

	root := &TreeNode{
		Name:     repoName,
		IsDir:    true,
		Children: make(map[string]*TreeNode),
	}

	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		curr := root
		for i, part := range parts {
			isLast := i == len(parts)-1
			if _, exists := curr.Children[part]; !exists {
				curr.Children[part] = &TreeNode{
					Name:     part,
					IsDir:    !isLast,
					Children: make(map[string]*TreeNode),
				}
			}
			curr = curr.Children[part]
		}
	}

	var sb strings.Builder
	sb.WriteString("Directory structure:\n└── " + repoName + "/\n")

	var renderTree func(node *TreeNode, prefix string)
	renderTree = func(node *TreeNode, prefix string) {
		var names []string
		for name := range node.Children {
			names = append(names, name)
		}
		sort.Strings(names)

		for i, name := range names {
			child := node.Children[name]
			isLast := i == len(names)-1
			connector := "├── "
			nextPrefix := prefix + "│   "
			if isLast {
				connector = "└── "
				nextPrefix = prefix + "    "
			}

			if child.IsDir {
				sb.WriteString(prefix + connector + child.Name + "/\n")
				renderTree(child, nextPrefix)
			} else {
				sb.WriteString(prefix + connector + child.Name + "\n")
			}
		}
	}

	renderTree(root, "    ")
	return sb.String()
}
