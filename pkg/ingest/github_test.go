package ingest

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		input       string
		wantOwner   string
		wantRepo    string
		wantBranch  string
		wantSubpath string
		wantErr     bool
	}{
		{
			input:     "https://github.com/charmbracelet/lipgloss",
			wantOwner: "charmbracelet",
			wantRepo:  "lipgloss",
		},
		{
			input:     "github.com/gin-gonic/gin",
			wantOwner: "gin-gonic",
			wantRepo:  "gin",
		},
		{
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			input:       "https://github.com/coderamp-labs/gitingest/tree/main/src",
			wantOwner:   "coderamp-labs",
			wantRepo:    "gitingest",
			wantBranch:  "main",
			wantSubpath: "src",
		},
		{
			input:       "https://github.com/anilpdv/UltraNav/Domain",
			wantOwner:   "anilpdv",
			wantRepo:    "UltraNav",
			wantSubpath: "Domain",
		},
		{
			input:       "owner/repo/sub/folder",
			wantOwner:   "owner",
			wantRepo:    "repo",
			wantSubpath: "sub/folder",
		},
		{
			input:   "invalid-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := ParseGitHubURL(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseGitHubURL(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseGitHubURL(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got.Owner != tt.wantOwner || got.Repo != tt.wantRepo {
			t.Errorf("ParseGitHubURL(%q) = owner:%s repo:%s, want owner:%s repo:%s", tt.input, got.Owner, got.Repo, tt.wantOwner, tt.wantRepo)
		}
		if tt.wantBranch != "" && got.Branch != tt.wantBranch {
			t.Errorf("ParseGitHubURL(%q) branch = %s, want %s", tt.input, got.Branch, tt.wantBranch)
		}
		if tt.wantSubpath != "" && got.Subpath != tt.wantSubpath {
			t.Errorf("ParseGitHubURL(%q) subpath = %s, want %s", tt.input, got.Subpath, tt.wantSubpath)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	text := "func main() {\n\tprintln(\"hello world\")\n}"
	tokens := EstimateTokens(text)
	if tokens <= 0 {
		t.Errorf("EstimateTokens(%q) = %d, want > 0", text, tokens)
	}
}
