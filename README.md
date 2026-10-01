# GitIngest Hub (Go Edition) ⚡

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](LICENSE)
[![UI](https://img.shields.io/badge/Style-Neubrutalism-FFE566?style=for-the-badge)](https://tailwindcss.com)

> Ultra-fast in-memory GitHub codebase ingestion, boundary chunker, and AST token optimizer with Chrome Built-in AI (Gemini Nano) summarization.

---

## ✨ Features

- **Zero-Disk Streaming**: Streams GitHub tarball archives straight into memory using Go `gzip` and `tar` decoders.
- **Smart Partitioning**: Divides massive repositories across non-overlapping, balanced chunks by file count, token limits, or KB size.
- **Subfolder / Subpath Ingestion**: Ingest targeted repository subdirectories (e.g. `owner/repo/Domain`).
- **AST Signature Extraction**: Minimizes token consumption by extracting types, protocols, interfaces, and signatures while eliding implementation bodies.
- **Lossless Token Minifier**: Strips redundant copyright headers and normalizes whitespace for ~20% token savings without losing code structure.
- **Deep Architecture Summarizer**: Performs multi-domain semantic clustering and architectural analysis.
- **Neubrutalism Visual Design**: High-contrast, accessibility-first design with responsive cards and 1-click clipboard integration.

---

## 🚀 Quick Start

### Run Locally

```bash
# Clone the repository
git clone https://github.com/anilpdv/gitingest-hub.git
cd gitingest-hub

# Build and run
go run main.go --port 8080
```

Visit `http://localhost:8080` in your browser.

---

## 🛠️ Tech Stack

- **Backend**: Go standard library (`net/http`, `go/parser`, `go/ast`)
- **Frontend**: HTMX, Tailwind CSS, Space Mono & Plus Jakarta Sans typography
- **AI Integration**: Chrome Built-in AI (Gemini Nano via `window.ai`)
- **Architecture**: In-memory streaming without temporary disk storage

---

## 📄 License

MIT License © 2024-2025 Anil Pdv

<!-- Co-authored documentation note -->
