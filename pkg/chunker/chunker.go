package chunker

import (
	"fmt"
	"strings"

	"gitingest-hub/pkg/model"
)

// FormatFileBlock creates the standard delimited file representation.
func FormatFileBlock(file model.IngestFile) string {
	var sb strings.Builder
	sb.WriteString("================================================\n")
	sb.WriteString(fmt.Sprintf("FILE: %s\n", file.Path))
	sb.WriteString("================================================\n")
	sb.WriteString(file.Content)
	if !strings.HasSuffix(file.Content, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func formatBytes(b int64) string {
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

// BuildChunkHeader creates a concise, ultra-compact header without wasting tokens repeating all file paths.
func BuildChunkHeader(repoName, branch string, chunkIdx, totalChunks, fileCount int, fullTree string) string {
	var sb strings.Builder
	sb.WriteString("================================================================================\n")
	sb.WriteString(fmt.Sprintf("# REPOSITORY: %s", repoName))
	if branch != "" {
		sb.WriteString(fmt.Sprintf(" (Branch: %s)", branch))
	}
	sb.WriteString("\n")

	if totalChunks > 1 {
		sb.WriteString(fmt.Sprintf("# CODEBASE PART: %d of %d (%d files)\n", chunkIdx, totalChunks, fileCount))
	} else {
		sb.WriteString(fmt.Sprintf("# FULL CODEBASE (%d files)\n", fileCount))
	}
	sb.WriteString("================================================================================\n\n")

	// If this is the first chunk or single file, optionally include the full directory tree
	if chunkIdx == 1 && fullTree != "" {
		sb.WriteString("# --- COMPLETE REPOSITORY DIRECTORY TREE ---\n")
		sb.WriteString(fullTree)
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("# --- BEGIN SOURCE CODE (PART %d of %d) ---\n\n", chunkIdx, totalChunks))
	return sb.String()
}

// SplitIntoChunks segments the repository files according to ONE chosen strategy.
func SplitIntoChunks(repoName, branch, tree string, files []model.IngestFile, opts model.IngestOptions) []model.Chunk {
	if len(files) == 0 {
		return nil
	}

	compressMode := opts.CompressMode
	if compressMode == "" {
		if opts.CompressForAI {
			compressMode = "compact"
		} else {
			compressMode = "none"
		}
	}

	switch opts.ChunkMode {
	case "parts":
		count := opts.ChunkCount
		if count <= 1 {
			return splitSingle(repoName, branch, tree, files, compressMode)
		}
		return splitByParts(repoName, branch, tree, files, count, compressMode)

	case "size_limit":
		limitKB := opts.SizeLimitKB
		if limitKB <= 0 {
			limitKB = 500 // default 500KB
		}
		return splitBySizeLimit(repoName, branch, tree, files, limitKB*1024, compressMode)

	case "token_limit":
		limit := opts.TokenLimit
		if limit <= 0 {
			limit = 50000 // default 50k tokens
		}
		return splitByTokenLimit(repoName, branch, tree, files, limit, compressMode)

	case "single":
		fallthrough
	default:
		return splitSingle(repoName, branch, tree, files, compressMode)
	}
}

// getFileFormattedWeight returns the exact byte size and token count of a file formatted in the target compressMode.
func getFileFormattedWeight(f model.IngestFile, compressMode string) (int, int) {
	block := FormatFileBlockWithOption(f, compressMode)
	bSize := len(block)
	tokenCount := f.TokenCount
	if compressMode != "none" {
		tokenCount = bSize / 4
		if tokenCount < 1 {
			tokenCount = 1
		}
	}
	return bSize, tokenCount
}

// splitSingle packs the entire repository into 1 single file.
func splitSingle(repoName, branch, tree string, files []model.IngestFile, compressMode string) []model.Chunk {
	header := BuildChunkHeader(repoName, branch, 1, 1, len(files), tree)
	var sb strings.Builder
	sb.WriteString(header)

	var filePaths []string
	totalTokens := 0

	for _, f := range files {
		block := FormatFileBlockWithOption(f, compressMode)
		sb.WriteString(block)
		filePaths = append(filePaths, f.Path)
		_, fTokens := getFileFormattedWeight(f, compressMode)
		totalTokens += fTokens
	}

	fullContent := sb.String()
	return []model.Chunk{
		{
			Index:       1,
			TotalChunks: 1,
			Title:       fmt.Sprintf("%s (Full Codebase - %d files)", repoName, len(files)),
			TokenCount:  totalTokens,
			ByteSize:    len(fullContent),
			FileCount:   len(files),
			Files:       filePaths,
			Content:     fullContent,
		},
	}
}

// splitByParts divides all repository files evenly across N distinct, non-overlapping chunks.
func splitByParts(repoName, branch, tree string, files []model.IngestFile, numParts int, compressMode string) []model.Chunk {
	if numParts <= 1 || len(files) <= 1 {
		return splitSingle(repoName, branch, tree, files, compressMode)
	}

	if numParts > len(files) {
		numParts = len(files)
	}

	treeOverhead := int64(len(tree) + 250)
	var totalBytes int64 = treeOverhead
	for _, f := range files {
		bSize, _ := getFileFormattedWeight(f, compressMode)
		totalBytes += int64(bSize)
	}
	targetBytesPerChunk := totalBytes / int64(numParts)
	if targetBytesPerChunk <= 0 {
		targetBytesPerChunk = 1
	}

	var buckets [][]model.IngestFile
	var currentBucket []model.IngestFile
	var currentBucketBytes int64 = treeOverhead // Chunk 1 includes tree overhead

	for i, f := range files {
		bSize, _ := getFileFormattedWeight(f, compressMode)
		fileWeight := int64(bSize)
		remainingFiles := len(files) - i
		remainingBuckets := numParts - len(buckets)

		// Must create new bucket if remaining files == remaining buckets to ensure we hit numParts
		if remainingBuckets > 1 && remainingFiles <= remainingBuckets {
			if len(currentBucket) > 0 {
				buckets = append(buckets, currentBucket)
			}
			currentBucket = []model.IngestFile{f}
			currentBucketBytes = fileWeight
			continue
		}

		currentBucket = append(currentBucket, f)
		currentBucketBytes += fileWeight

		// If current bucket exceeded target and we still have buckets to fill
		if remainingBuckets > 1 && currentBucketBytes >= targetBytesPerChunk && len(currentBucket) > 0 {
			buckets = append(buckets, currentBucket)
			currentBucket = nil
			currentBucketBytes = 0
		}
	}
	if len(currentBucket) > 0 {
		buckets = append(buckets, currentBucket)
	}

	totalParts := len(buckets)
	var results []model.Chunk

	for i, bucket := range buckets {
		header := BuildChunkHeader(repoName, branch, i+1, totalParts, len(bucket), tree)
		var sb strings.Builder
		sb.WriteString(header)

		var filePaths []string
		chunkTokens := 0

		for _, f := range bucket {
			block := FormatFileBlockWithOption(f, compressMode)
			sb.WriteString(block)
			filePaths = append(filePaths, f.Path)
			_, fTokens := getFileFormattedWeight(f, compressMode)
			chunkTokens += fTokens
		}

		fullContent := sb.String()
		title := fmt.Sprintf("Part %d of %d (%d files: %s ... %s)", 
			i+1, totalParts, len(bucket), bucket[0].Path, bucket[len(bucket)-1].Path)

		results = append(results, model.Chunk{
			Index:       i + 1,
			TotalChunks: totalParts,
			Title:       title,
			TokenCount:  chunkTokens,
			ByteSize:    len(fullContent),
			FileCount:   len(bucket),
			Files:       filePaths,
			Content:     fullContent,
		})
	}

	return results
}

// splitBySizeLimit groups files into chunks not exceeding the specified byte limit.
func splitBySizeLimit(repoName, branch, tree string, files []model.IngestFile, maxBytes int, compressMode string) []model.Chunk {
	if maxBytes <= 0 {
		return splitSingle(repoName, branch, tree, files, compressMode)
	}

	treeOverhead := len(tree) + 250
	var groups [][]model.IngestFile
	var currentGroup []model.IngestFile
	currentBytes := treeOverhead // Chunk 1 includes tree overhead

	for _, f := range files {
		fileByteSize, _ := getFileFormattedWeight(f, compressMode)
		if len(currentGroup) > 0 && currentBytes+fileByteSize > maxBytes {
			groups = append(groups, currentGroup)
			currentGroup = []model.IngestFile{f}
			currentBytes = fileByteSize
		} else {
			currentGroup = append(currentGroup, f)
			currentBytes += fileByteSize
		}
	}
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	totalParts := len(groups)
	var results []model.Chunk

	for i, group := range groups {
		header := BuildChunkHeader(repoName, branch, i+1, totalParts, len(group), tree)
		var sb strings.Builder
		sb.WriteString(header)

		var filePaths []string
		chunkTokens := 0

		for _, f := range group {
			block := FormatFileBlockWithOption(f, compressMode)
			sb.WriteString(block)
			filePaths = append(filePaths, f.Path)
			_, fTokens := getFileFormattedWeight(f, compressMode)
			chunkTokens += fTokens
		}

		fullContent := sb.String()
		title := fmt.Sprintf("Size Chunk %d of %d (%d files: %s ... %s)", 
			i+1, totalParts, len(group), group[0].Path, group[len(group)-1].Path)

		results = append(results, model.Chunk{
			Index:       i + 1,
			TotalChunks: totalParts,
			Title:       title,
			TokenCount:  chunkTokens,
			ByteSize:    len(fullContent),
			FileCount:   len(group),
			Files:       filePaths,
			Content:     fullContent,
		})
	}

	return results
}

// splitByTokenLimit groups files into chunks not exceeding the specified token threshold.
func splitByTokenLimit(repoName, branch, tree string, files []model.IngestFile, maxTokens int, compressMode string) []model.Chunk {
	if maxTokens <= 0 {
		return splitSingle(repoName, branch, tree, files, compressMode)
	}

	var groups [][]model.IngestFile
	var currentGroup []model.IngestFile
	currentTokens := 0

	for _, f := range files {
		_, fTokens := getFileFormattedWeight(f, compressMode)
		if len(currentGroup) > 0 && currentTokens+fTokens > maxTokens {
			groups = append(groups, currentGroup)
			currentGroup = []model.IngestFile{f}
			currentTokens = fTokens
		} else {
			currentGroup = append(currentGroup, f)
			currentTokens += fTokens
		}
	}
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	totalParts := len(groups)
	var results []model.Chunk

	for i, group := range groups {
		header := BuildChunkHeader(repoName, branch, i+1, totalParts, len(group), tree)
		var sb strings.Builder
		sb.WriteString(header)

		var filePaths []string
		chunkTokens := 0

		for _, f := range group {
			block := FormatFileBlockWithOption(f, compressMode)
			sb.WriteString(block)
			filePaths = append(filePaths, f.Path)
			_, fTokens := getFileFormattedWeight(f, compressMode)
			chunkTokens += fTokens
		}

		fullContent := sb.String()
		title := fmt.Sprintf("Token Chunk %d of %d (%d files: %s ... %s)", 
			i+1, totalParts, len(group), group[0].Path, group[len(group)-1].Path)

		results = append(results, model.Chunk{
			Index:       i + 1,
			TotalChunks: totalParts,
			Title:       title,
			TokenCount:  chunkTokens,
			ByteSize:    len(fullContent),
			FileCount:   len(group),
			Files:       filePaths,
			Content:     fullContent,
		})
	}

	return results
}
