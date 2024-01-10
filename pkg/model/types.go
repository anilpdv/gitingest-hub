package model

import "time"

// IngestFile represents a single extracted file from a repository.
type IngestFile struct {
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Content    string `json:"content"`
	TokenCount int    `json:"token_count"`
}

// IngestOptions defines configuration for repository ingestion and chunking.
type IngestOptions struct {
	RepoURL      string `json:"repo_url"`
	Branch       string `json:"branch"`
	Subpath      string `json:"subpath"`
	IncludeDocs  bool   `json:"include_docs"`
	IncludeTests bool   `json:"include_tests"`
	ChunkMode     string `json:"chunk_mode"` // "single", "parts", "token_limit", "size_limit"
	ChunkCount    int    `json:"chunk_count"` // e.g. 1 to 50
	TokenLimit    int    `json:"token_limit"` // e.g. 50000
	SizeLimitKB   int    `json:"size_limit_kb"` // e.g. 250, 500, 1000 KB
	CodeScope     string `json:"code_scope"`  // "full" or "ast_signatures"
	MinifyTokens  bool   `json:"minify_tokens"` // Lossless whitespace & boilerplate minifier (Default: true)
	CompressForAI bool   `json:"compress_for_ai"` // Minify/pack code, strip license boilerplate & compact empty lines
	CompressMode  string `json:"compress_mode"`  // "none", "compact", "ast_signatures"
}

// Chunk represents an individual chunk of the codebase with full file boundary preservation.
type Chunk struct {
	Index       int      `json:"index"`
	TotalChunks int      `json:"total_chunks"`
	Title       string   `json:"title"`
	TokenCount  int      `json:"token_count"`
	ByteSize    int      `json:"byte_size"`
	FileCount   int      `json:"file_count"`
	Files       []string `json:"files"`
	Content     string   `json:"content"`
	Summary     string   `json:"summary,omitempty"`
}

// IngestResult holds the full repository ingestion state and generated chunks.
type IngestResult struct {
	ID             string        `json:"id"`
	RepoName       string        `json:"repo_name"`
	RepoURL        string        `json:"repo_url"`
	Branch         string        `json:"branch"`
	TotalFiles     int           `json:"total_files"`
	TotalTokens    int           `json:"total_tokens"`
	TotalBytes     int64         `json:"total_bytes"`
	SavedBytes     int64         `json:"saved_bytes"`
	SavedPercent   int           `json:"saved_percent"`
	DirectoryTree  string        `json:"directory_tree"`
	Files          []IngestFile  `json:"files"`
	Chunks         []Chunk       `json:"chunks"`
	Options        IngestOptions `json:"options"`
	ProcessingTime time.Duration `json:"processing_time"`
	CreatedAt      time.Time     `json:"created_at"`
}
