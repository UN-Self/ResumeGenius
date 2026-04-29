package parsing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxReadmePreviewLength = 1800
	maxTopLevelEntries     = 10
)

type gitCommandRunner func(dir string, args ...string) ([]byte, error)

type GitRepositoryExtractor struct {
	makeTempDir func(dir, pattern string) (string, error)
	removeAll   func(path string) error
	readFile    func(path string) ([]byte, error)
	readDir     func(name string) ([]os.DirEntry, error)
	runGit      gitCommandRunner
}

func NewGitExtractor() *GitRepositoryExtractor {
	return &GitRepositoryExtractor{
		makeTempDir: os.MkdirTemp,
		removeAll:   os.RemoveAll,
		readFile:    os.ReadFile,
		readDir:     os.ReadDir,
		runGit:      defaultRunGitCommand,
	}
}

func (g *GitRepositoryExtractor) Extract(repoURL string) (*ParsedContent, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil, ErrAssetURIMissing
	}

	repoDir, cleanup, err := g.prepareRepository(repoURL)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	summary, err := g.buildRepositorySummary(repoURL, repoDir)
	if err != nil {
		return nil, err
	}

	return &ParsedContent{
		Text: summary,
	}, nil
}

func (g *GitRepositoryExtractor) prepareRepository(repoURL string) (string, func(), error) {
	if localPath, ok := resolveLocalRepositoryPath(repoURL); ok {
		info, err := os.Stat(localPath)
		if err != nil {
			return "", nil, fmt.Errorf("stat local repository: %w", err)
		}
		if !info.IsDir() {
			return "", nil, fmt.Errorf("local repository path is not a directory: %s", localPath)
		}
		return localPath, func() {}, nil
	}

	tempDir, err := g.makeTempDir("", "resume-genius-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir for repository clone: %w", err)
	}

	repoDir := filepath.Join(tempDir, "repo")
	if _, err := g.runGit("", "clone", "--depth", "1", repoURL, repoDir); err != nil {
		_ = g.removeAll(tempDir)
		return "", nil, fmt.Errorf("clone repository: %w", err)
	}

	return repoDir, func() {
		_ = g.removeAll(tempDir)
	}, nil
}

func (g *GitRepositoryExtractor) buildRepositorySummary(repoURL, repoDir string) (string, error) {
	repoName := inferRepositoryName(repoURL, repoDir)
	readmePreview, err := g.readReadmePreview(repoDir)
	if err != nil {
		return "", err
	}

	techStack, err := g.detectTechStack(repoDir)
	if err != nil {
		return "", err
	}

	structure, err := g.summarizeTopLevelStructure(repoDir)
	if err != nil {
		return "", err
	}

	parts := []string{
		fmt.Sprintf("Repository: %s", repoName),
		fmt.Sprintf("Source: %s", repoURL),
	}
	if readmePreview != "" {
		parts = append(parts, "README:\n"+readmePreview)
	}
	if len(techStack) > 0 {
		parts = append(parts, "Tech stack: "+strings.Join(techStack, ", "))
	}
	if len(structure) > 0 {
		parts = append(parts, "Top-level structure: "+strings.Join(structure, ", "))
	}

	return strings.Join(parts, "\n\n"), nil
}

func (g *GitRepositoryExtractor) readReadmePreview(repoDir string) (string, error) {
	candidates := []string{
		"README.md",
		"README",
		"readme.md",
		"readme",
		"Readme.md",
	}

	for _, name := range candidates {
		path := filepath.Join(repoDir, name)
		content, err := g.readFile(path)
		if err == nil {
			return truncateAndNormalizeText(string(content), maxReadmePreviewLength), nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("read repository readme: %w", err)
		}
	}

	return "", nil
}

func (g *GitRepositoryExtractor) detectTechStack(repoDir string) ([]string, error) {
	labels := map[string]bool{}
	mark := func(label string) {
		if strings.TrimSpace(label) != "" {
			labels[label] = true
		}
	}

	markIfExists := func(name, label string) error {
		_, err := os.Stat(filepath.Join(repoDir, name))
		if err == nil {
			mark(label)
			return nil
		}
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", name, err)
	}

	for _, pair := range []struct {
		name  string
		label string
	}{
		{"go.mod", "Go"},
		{"tsconfig.json", "TypeScript"},
		{"requirements.txt", "Python"},
		{"pyproject.toml", "Python"},
		{"Cargo.toml", "Rust"},
		{"pom.xml", "Java"},
		{"build.gradle", "Java"},
		{"build.gradle.kts", "Java"},
		{"Gemfile", "Ruby"},
		{"composer.json", "PHP"},
		{"Dockerfile", "Docker"},
		{"docker-compose.yml", "Docker Compose"},
		{"docker-compose.yaml", "Docker Compose"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "Yarn"},
	} {
		if err := markIfExists(pair.name, pair.label); err != nil {
			return nil, err
		}
	}

	if err := g.detectPackageJSONTech(repoDir, mark); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(labels))
	for label := range labels {
		result = append(result, label)
	}
	sort.Strings(result)
	return result, nil
}

func (g *GitRepositoryExtractor) detectPackageJSONTech(repoDir string, mark func(string)) error {
	path := filepath.Join(repoDir, "package.json")
	content, err := g.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read package.json: %w", err)
	}

	mark("Node.js")
	mark("JavaScript")

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil
	}

	allDeps := make(map[string]string, len(pkg.Dependencies)+len(pkg.DevDependencies))
	for name, version := range pkg.Dependencies {
		allDeps[strings.ToLower(name)] = version
	}
	for name, version := range pkg.DevDependencies {
		allDeps[strings.ToLower(name)] = version
	}

	if _, ok := allDeps["typescript"]; ok {
		mark("TypeScript")
	}
	if _, ok := allDeps["react"]; ok {
		mark("React")
	}
	if _, ok := allDeps["next"]; ok {
		mark("Next.js")
	}
	if _, ok := allDeps["vue"]; ok {
		mark("Vue")
	}
	if _, ok := allDeps["svelte"]; ok {
		mark("Svelte")
	}
	if _, ok := allDeps["vite"]; ok {
		mark("Vite")
	}
	if _, ok := allDeps["tailwindcss"]; ok {
		mark("Tailwind CSS")
	}
	if _, ok := allDeps["@angular/core"]; ok {
		mark("Angular")
	}

	return nil
}

func (g *GitRepositoryExtractor) summarizeTopLevelStructure(repoDir string) ([]string, error) {
	entries, err := g.readDir(repoDir)
	if err != nil {
		return nil, fmt.Errorf("read repository root: %w", err)
	}

	filtered := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" || name == "node_modules" || name == "vendor" {
			continue
		}

		if entry.IsDir() {
			filtered = append(filtered, name+"/")
		} else {
			filtered = append(filtered, name)
		}
	}

	sort.Strings(filtered)
	if len(filtered) > maxTopLevelEntries {
		filtered = filtered[:maxTopLevelEntries]
	}

	return filtered, nil
}

func resolveLocalRepositoryPath(repoURL string) (string, bool) {
	if info, err := os.Stat(repoURL); err == nil && info.IsDir() {
		return repoURL, true
	}

	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "file" {
		return "", false
	}

	path := parsed.Path
	if parsed.Host != "" {
		path = "//" + parsed.Host + parsed.Path
	}
	path = filepath.FromSlash(path)
	if len(path) >= 3 && (path[0] == '\\' || path[0] == '/') && path[2] == ':' {
		path = path[1:]
	}
	return path, true
}

func inferRepositoryName(repoURL, repoDir string) string {
	if base := filepath.Base(repoDir); base != "" && base != "." && base != string(filepath.Separator) {
		if base != "repo" {
			return base
		}
	}

	trimmed := strings.TrimSuffix(strings.TrimSpace(repoURL), "/")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	if trimmed == "" {
		return "unknown"
	}

	if parsed, err := url.Parse(trimmed); err == nil && parsed.Path != "" {
		segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(segments) > 0 && segments[len(segments)-1] != "" {
			return segments[len(segments)-1]
		}
	}

	return filepath.Base(trimmed)
}

func truncateAndNormalizeText(input string, limit int) string {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return ""
	}

	if len(normalized) <= limit {
		return normalized
	}
	return strings.TrimSpace(normalized[:limit]) + "..."
}

func defaultRunGitCommand(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("%w: %s", err, message)
	}

	return stdout.Bytes(), nil
}
