package parsing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitExtractorExtract_SummarizesLocalRepository(t *testing.T) {
	repoDir := t.TempDir()
	writeTestFile(t, filepath.Join(repoDir, "README.md"), "# ResumeGenius\nA resume builder for modern job applications.\n")
	writeTestFile(t, filepath.Join(repoDir, "go.mod"), "module example.com/resumegenius\n\ngo 1.25.0\n")
	writeTestFile(t, filepath.Join(repoDir, "package.json"), `{
  "name": "resumegenius",
  "dependencies": {
    "react": "^19.0.0",
    "vite": "^6.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}`)
	writeTestFile(t, filepath.Join(repoDir, "Dockerfile"), "FROM node:20-alpine\n")
	if err := os.MkdirAll(filepath.Join(repoDir, "backend"), 0755); err != nil {
		t.Fatalf("mkdir backend: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, "frontend"), 0755); err != nil {
		t.Fatalf("mkdir frontend: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, "docs"), 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}

	extractor := NewGitExtractor()
	parsed, err := extractor.Extract(repoDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed git content")
	}

	for _, want := range []string{
		"Repository: " + filepath.Base(repoDir),
		"README:\n# ResumeGenius",
		"Tech stack: ",
		"Go",
		"Node.js",
		"React",
		"TypeScript",
		"Vite",
		"Docker",
		"Top-level structure: ",
		"backend/",
		"frontend/",
		"docs/",
		"README.md",
	} {
		if !strings.Contains(parsed.Text, want) {
			t.Fatalf("expected parsed text to contain %q, got %q", want, parsed.Text)
		}
	}
}

func TestGitExtractorExtract_ClonesRemoteRepositoryWhenNeeded(t *testing.T) {
	tempRoot := t.TempDir()
	var capturedArgs []string

	extractor := &GitRepositoryExtractor{
		makeTempDir: func(dir, pattern string) (string, error) {
			return tempRoot, nil
		},
		removeAll: os.RemoveAll,
		readFile:  os.ReadFile,
		readDir:   os.ReadDir,
		runGit: func(dir string, args ...string) ([]byte, error) {
			capturedArgs = append([]string(nil), args...)
			repoDir := filepath.Join(tempRoot, "repo")
			if err := os.MkdirAll(repoDir, 0755); err != nil {
				t.Fatalf("mkdir repo dir: %v", err)
			}
			writeTestFile(t, filepath.Join(repoDir, "README.md"), "Remote repository\n")
			return []byte("ok"), nil
		},
	}

	parsed, err := extractor.Extract("https://github.com/example/project.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed git content")
	}
	if len(capturedArgs) < 4 {
		t.Fatalf("expected git clone args to be captured, got %+v", capturedArgs)
	}
	if capturedArgs[0] != "clone" {
		t.Fatalf("expected git clone invocation, got %+v", capturedArgs)
	}
	if !strings.Contains(parsed.Text, "Repository: project") {
		t.Fatalf("expected repository name derived from url, got %q", parsed.Text)
	}
	if !strings.Contains(parsed.Text, "README:\nRemote repository") {
		t.Fatalf("expected remote readme in summary, got %q", parsed.Text)
	}
}

func TestGitExtractorExtract_ReturnsErrorForBlankRepositoryURL(t *testing.T) {
	extractor := NewGitExtractor()

	_, err := extractor.Extract("   ")
	if err == nil {
		t.Fatal("expected blank repository url error")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir parent dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
