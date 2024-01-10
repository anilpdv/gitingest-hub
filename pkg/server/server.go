package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitingest-hub/pkg/chunker"
	"gitingest-hub/pkg/ingest"
	"gitingest-hub/pkg/model"
)

type CachedSession struct {
	Result    model.IngestResult
	Files     []model.IngestFile
	ExpiresAt time.Time
}

type Server struct {
	templates *template.Template
	cache     map[string]*CachedSession
	mu        sync.RWMutex
}

func formatBytes(v interface{}) string {
	var b int64
	switch val := v.(type) {
	case int:
		b = int64(val)
	case int64:
		b = val
	case int32:
		b = int64(val)
	case uint64:
		b = int64(val)
	default:
		return fmt.Sprintf("%v B", v)
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (s *Server) cleanupCacheWorker() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, sess := range s.cache {
			if now.After(sess.ExpiresAt) {
				delete(s.cache, id)
			}
		}
		s.mu.Unlock()
	}
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func NewServer() (*Server, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"formatBytes": formatBytes,
	}).ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	s := &Server{
		templates: tmpl,
		cache:     make(map[string]*CachedSession),
	}

	// Clean up old cache entries periodically
	go s.cleanupCacheWorker()

	return s, nil
}

// HandleIndex serves the main dashboard.
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.templates.ExecuteTemplate(w, "index.html", nil)
}

// HandleIngest handles GitHub repository fetching and initial chunking.
func (s *Server) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.renderError(w, "Invalid form data")
		return
	}

	repoURL := strings.TrimSpace(r.FormValue("repo_url"))
	if repoURL == "" {
		s.renderError(w, "GitHub repository URL cannot be empty.")
		return
	}

	branch := strings.TrimSpace(r.FormValue("branch"))
	chunkCountStr := r.FormValue("chunk_count")
	chunkMode := r.FormValue("chunk_mode")
	tokenLimitStr := r.FormValue("token_limit")
	sizeLimitStr := r.FormValue("size_limit_kb")

	chunkCount, _ := strconv.Atoi(chunkCountStr)
	tokenLimit, _ := strconv.Atoi(tokenLimitStr)
	sizeLimitKB, _ := strconv.Atoi(sizeLimitStr)

	if chunkCount <= 0 {
		chunkCount = 1
	}

	codeScope := r.FormValue("code_scope")
	if codeScope == "" {
		codeScope = "full"
	}

	minifyTokens := r.FormValue("minify_tokens") != "0" && r.FormValue("minify_tokens") != "false"
	compressMode := r.FormValue("compress_mode")

	if compressMode == "" {
		if codeScope == "ast_signatures" {
			compressMode = "ast_signatures"
		} else if minifyTokens {
			compressMode = "compact"
		} else {
			compressMode = "none"
		}
	} else if compressMode == "ast_signatures" {
		codeScope = "ast_signatures"
	} else if compressMode == "compact" {
		codeScope = "full"
		minifyTokens = true
	} else if compressMode == "none" {
		codeScope = "full"
		minifyTokens = false
	}

	opts := model.IngestOptions{
		RepoURL:       repoURL,
		Branch:        branch,
		ChunkMode:     chunkMode,
		ChunkCount:    chunkCount,
		TokenLimit:    tokenLimit,
		SizeLimitKB:   sizeLimitKB,
		CodeScope:     codeScope,
		MinifyTokens:  minifyTokens,
		CompressMode:  compressMode,
		CompressForAI: compressMode != "none",
	}

	if opts.ChunkMode == "" {
		if chunkCount == 1 {
			opts.ChunkMode = "single"
		} else {
			opts.ChunkMode = "parts"
		}
	}

	startTime := time.Now()
	files, parsed, err := ingest.FetchRepository(opts)
	if err != nil {
		s.renderError(w, err.Error())
		return
	}
	duration := time.Since(startTime)

	repoName := fmt.Sprintf("%s/%s", parsed.Owner, parsed.Repo)
	if parsed.Subpath != "" {
		repoName = fmt.Sprintf("%s/%s/%s", parsed.Owner, parsed.Repo, parsed.Subpath)
	}
	tree := ingest.GenerateDirectoryTree(repoName, files)

	var totalTokens int
	var totalBytes int64
	for _, f := range files {
		totalTokens += f.TokenCount
		totalBytes += f.Size
	}

	chunks := chunker.SplitIntoChunks(repoName, branch, tree, files, opts)

	var savedBytes int64
	var savedPercent int
	if opts.CompressMode != "none" {
		var origContentBytes int64
		var compContentBytes int64
		for _, f := range files {
			origContentBytes += int64(len(f.Content))
			compContentBytes += int64(len(chunker.ProcessFileContent(f, opts.CompressMode)))
		}
		if origContentBytes > compContentBytes {
			savedBytes = origContentBytes - compContentBytes
			savedPercent = int((float64(savedBytes) / float64(origContentBytes)) * 100)
		}
	}

	sessionID := generateID()
	sourceURL := fmt.Sprintf("https://github.com/%s/%s", parsed.Owner, parsed.Repo)
	if parsed.Subpath != "" {
		branchPath := branch
		if branchPath == "" || branchPath == "HEAD" {
			branchPath = "tree/main"
		} else {
			branchPath = "tree/" + branchPath
		}
		sourceURL = fmt.Sprintf("https://github.com/%s/%s/%s/%s", parsed.Owner, parsed.Repo, branchPath, parsed.Subpath)
	}

	result := model.IngestResult{
		ID:             sessionID,
		RepoName:       repoName,
		RepoURL:        sourceURL,
		Branch:         branch,
		TotalFiles:     len(files),
		TotalTokens:    totalTokens,
		TotalBytes:     totalBytes,
		SavedBytes:     savedBytes,
		SavedPercent:   savedPercent,
		DirectoryTree:  tree,
		Files:          files,
		Chunks:         chunks,
		Options:        opts,
		ProcessingTime: duration.Round(time.Millisecond),
		CreatedAt:      time.Now(),
	}

	// Cache session for fast instant re-chunking
	s.mu.Lock()
	s.cache[sessionID] = &CachedSession{
		Result:    result,
		Files:     files,
		ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	s.mu.Unlock()

	data := struct {
		Result         model.IngestResult
		FormattedBytes string
		Error          string
	}{
		Result:         result,
		FormattedBytes: formatBytes(totalBytes),
	}

	s.templates.ExecuteTemplate(w, "chunks.html", data)
}

// HandleRechunk dynamically re-splits cached repository files without network calls.
func (s *Server) HandleRechunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.renderError(w, "Invalid form data")
		return
	}

	sessionID := r.FormValue("session_id")
	s.mu.RLock()
	cached, exists := s.cache[sessionID]
	s.mu.RUnlock()

	if !exists {
		s.renderError(w, "Session expired or not found. Please re-ingest the repository.")
		return
	}

	chunkCountStr := r.FormValue("chunk_count")
	chunkMode := r.FormValue("chunk_mode")
	tokenLimitStr := r.FormValue("token_limit")
	sizeLimitStr := r.FormValue("size_limit_kb")
	codeScope := r.FormValue("code_scope")
	if codeScope == "" {
		codeScope = cached.Result.Options.CodeScope
		if codeScope == "" {
			codeScope = "full"
		}
	}

	minifyTokens := true
	if r.FormValue("minify_tokens") == "0" || r.FormValue("minify_tokens") == "false" {
		minifyTokens = false
	} else if r.FormValue("minify_tokens") == "1" || r.FormValue("minify_tokens") == "true" {
		minifyTokens = true
	} else {
		minifyTokens = cached.Result.Options.MinifyTokens
	}

	compressMode := r.FormValue("compress_mode")
	if compressMode == "" {
		if codeScope == "ast_signatures" {
			compressMode = "ast_signatures"
		} else if minifyTokens {
			compressMode = "compact"
		} else {
			compressMode = "none"
		}
	} else if compressMode == "ast_signatures" {
		codeScope = "ast_signatures"
	} else if compressMode == "compact" {
		codeScope = "full"
		minifyTokens = true
	} else if compressMode == "none" {
		codeScope = "full"
		minifyTokens = false
	}

	chunkCount, _ := strconv.Atoi(chunkCountStr)
	tokenLimit, _ := strconv.Atoi(tokenLimitStr)
	sizeLimitKB, _ := strconv.Atoi(sizeLimitStr)

	opts := cached.Result.Options
	opts.CodeScope = codeScope
	opts.MinifyTokens = minifyTokens
	opts.CompressMode = compressMode
	opts.CompressForAI = compressMode != "none"

	if chunkMode == "token_limit" && tokenLimit > 0 {
		opts.ChunkMode = "token_limit"
		opts.TokenLimit = tokenLimit
	} else if chunkMode == "size_limit" && sizeLimitKB > 0 {
		opts.ChunkMode = "size_limit"
		opts.SizeLimitKB = sizeLimitKB
	} else if chunkCount > 0 {
		opts.ChunkCount = chunkCount
		if chunkCount == 1 {
			opts.ChunkMode = "single"
		} else {
			opts.ChunkMode = "parts"
		}
	}

	newChunks := chunker.SplitIntoChunks(cached.Result.RepoName, cached.Result.Branch, cached.Result.DirectoryTree, cached.Files, opts)
	cached.Result.Chunks = newChunks
	cached.Result.Options = opts

	var savedBytes int64
	var savedPercent int
	if opts.CompressMode != "none" {
		var origContentBytes int64
		var compContentBytes int64
		for _, f := range cached.Files {
			origContentBytes += int64(len(f.Content))
			compContentBytes += int64(len(chunker.ProcessFileContent(f, opts.CompressMode)))
		}
		if origContentBytes > compContentBytes {
			savedBytes = origContentBytes - compContentBytes
			savedPercent = int((float64(savedBytes) / float64(origContentBytes)) * 100)
		}
	}
	cached.Result.SavedBytes = savedBytes
	cached.Result.SavedPercent = savedPercent

	data := struct {
		Result         model.IngestResult
		FormattedBytes string
		Error          string
	}{
		Result:         cached.Result,
		FormattedBytes: formatBytes(cached.Result.TotalBytes),
	}

	s.templates.ExecuteTemplate(w, "chunks.html", data)
}

// HandleDownload streams an individual chunk file as a .txt download.
func (s *Server) HandleDownload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	chunkIndexStr := r.URL.Query().Get("chunk_index")
	chunkIndex, _ := strconv.Atoi(chunkIndexStr)

	s.mu.RLock()
	cached, exists := s.cache[sessionID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	var targetChunk *model.Chunk
	for i := range cached.Result.Chunks {
		if cached.Result.Chunks[i].Index == chunkIndex {
			targetChunk = &cached.Result.Chunks[i]
			break
		}
	}

	if targetChunk == nil {
		http.Error(w, "Chunk not found", http.StatusNotFound)
		return
	}

	cleanRepo := strings.ReplaceAll(cached.Result.RepoName, "/", "_")
	filename := fmt.Sprintf("%s_chunk_%d_of_%d.txt", cleanRepo, targetChunk.Index, targetChunk.TotalChunks)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(targetChunk.Content)))
	w.Write([]byte(targetChunk.Content))
}

func (s *Server) renderError(w http.ResponseWriter, msg string) {
	data := struct {
		Result         model.IngestResult
		FormattedBytes string
		Error          string
	}{
		Error: msg,
	}
	s.templates.ExecuteTemplate(w, "chunks.html", data)
}
