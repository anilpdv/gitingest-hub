package chunker

import (
	"strings"
	"testing"

	"gitingest-hub/pkg/model"
)

func TestSplitIntoChunks(t *testing.T) {
	sampleFiles := []model.IngestFile{
		{Path: "main.go", Content: "package main\nfunc main() {}", TokenCount: 100, Size: 30},
		{Path: "pkg/api.go", Content: "package pkg\nfunc Handler() {}", TokenCount: 200, Size: 40},
		{Path: "pkg/db.go", Content: "package pkg\nfunc Connect() {}", TokenCount: 300, Size: 50},
		{Path: "pkg/model.go", Content: "package pkg\ntype User struct {}", TokenCount: 150, Size: 35},
	}

	tree := "Directory structure:\n└── repo/\n    ├── main.go\n    └── pkg/\n        ├── api.go\n        ├── db.go\n        └── model.go\n"

	t.Run("Default Single Chunk", func(t *testing.T) {
		chunks := SplitIntoChunks("myrepo", "main", tree, sampleFiles, model.IngestOptions{
			ChunkMode: "single",
		})

		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].FileCount != 4 {
			t.Errorf("expected 4 files, got %d", chunks[0].FileCount)
		}
		if !strings.Contains(chunks[0].Content, "FILE: main.go") {
			t.Errorf("expected chunk to contain main.go")
		}
	})

	t.Run("Split into 2 Parts", func(t *testing.T) {
		chunks := SplitIntoChunks("myrepo", "main", tree, sampleFiles, model.IngestOptions{
			ChunkMode:  "parts",
			ChunkCount: 2,
		})

		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d", len(chunks))
		}
		totalFiles := chunks[0].FileCount + chunks[1].FileCount
		if totalFiles != 4 {
			t.Errorf("expected total 4 files across chunks, got %d", totalFiles)
		}
		// Ensure non-overlapping files
		if chunks[0].Files[0] == chunks[1].Files[0] {
			t.Errorf("chunks must contain distinct files!")
		}
	})

	t.Run("Split by Token Limit", func(t *testing.T) {
		chunks := SplitIntoChunks("myrepo", "main", tree, sampleFiles, model.IngestOptions{
			ChunkMode:  "token_limit",
			TokenLimit: 350,
		})

		if len(chunks) < 2 {
			t.Fatalf("expected at least 2 chunks for 350 token limit, got %d", len(chunks))
		}
		for i, c := range chunks {
			if len(c.Files) == 0 {
				t.Errorf("chunk %d has 0 files", i)
			}
		}
	})

	t.Run("AI Compression and Boilerplate Stripping", func(t *testing.T) {
		codeWithBoilerplate := `/*
 * Copyright (c) 2024 Acme Corp.
 * All rights reserved.
 * SPDX-License-Identifier: MIT
 */

package main



func Run() string {   
	return "ok"   
}
`
		compressed := CompressContentForAI(codeWithBoilerplate)
		if strings.Contains(compressed, "Copyright") {
			t.Errorf("expected boilerplate license to be stripped")
		}
		if strings.Contains(compressed, "\n\n\n") {
			t.Errorf("expected 3+ newlines to be compacted")
		}
		if !strings.Contains(compressed, `func Run() string {`) {
			t.Errorf("expected actual code logic to be preserved")
		}

		chunks := SplitIntoChunks("myrepo", "main", tree, []model.IngestFile{
			{Path: "main.go", Content: codeWithBoilerplate, TokenCount: 80, Size: int64(len(codeWithBoilerplate))},
		}, model.IngestOptions{
			ChunkMode:     "single",
			CompressForAI: true,
		})

		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if !strings.Contains(chunks[0].Content, `<file path="main.go">`) {
			t.Errorf("expected AI XML file tag when CompressForAI is active")
		}
	})

	t.Run("AST Signature Extraction - Go, Python, TypeScript", func(t *testing.T) {
		goCode := `package main

type Config struct {
	Port int
	Host string
}

func StartServer(cfg Config) error {
	// complex database connections and routing logic
	x := 10
	for i := 0; i < x; i++ {
		println(i)
	}
	return nil
}
`
		goExtracted := ExtractGoASTSignatures(goCode)
		if !strings.Contains(goExtracted, "type Config struct") {
			t.Errorf("expected Config struct to be preserved")
		}
		if !strings.Contains(goExtracted, "func StartServer(cfg Config) error") {
			t.Errorf("expected StartServer signature to be preserved")
		}
		if strings.Contains(goExtracted, "for i := 0") {
			t.Errorf("expected internal loop to be elided")
		}

		pyCode := `import os
from typing import List

class DataService:
    """Service handling data retrieval"""
    def __init__(self, api_key: str):
        self.api_key = api_key

    def fetch_records(self, limit: int = 100) -> List[dict]:
        # raw database queries and caching
        res = []
        for i in range(limit):
            res.append({"id": i})
        return res
`
		pyExtracted := ExtractPythonSignatures(pyCode)
		if !strings.Contains(pyExtracted, "class DataService:") {
			t.Errorf("expected class header to be preserved")
		}
		if !strings.Contains(pyExtracted, "def fetch_records(self, limit: int = 100) -> List[dict]:") {
			t.Errorf("expected fetch_records method signature to be preserved")
		}
		if strings.Contains(pyExtracted, "for i in range(limit):") {
			t.Errorf("expected Python loop body to be elided")
		}

		tsCode := `import { Request, Response } from 'express';

export interface UserDTO {
	id: string;
	email: string;
}

export function handleUser(req: Request, res: Response): Promise<UserDTO> {
	const user = { id: '1', email: 'test@example.com' };
	return Promise.resolve(user);
}
`
		tsExtracted := ExtractTypeScriptSignatures(tsCode)
		t.Logf("tsExtracted:\n%s", tsExtracted)
		if !strings.Contains(tsExtracted, "export interface UserDTO") {
			t.Errorf("expected TS interface to be preserved")
		}
		if !strings.Contains(tsExtracted, "export function handleUser(req: Request, res: Response): Promise<UserDTO>") {
			t.Errorf("expected TS function signature to be preserved")
		}
		if strings.Contains(tsExtracted, "const user =") {
			t.Errorf("expected TS internal variables to be elided")
		}
	})
}
