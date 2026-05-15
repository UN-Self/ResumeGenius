package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
	"gorm.io/gorm"
)

// SSHKeyProvider abstracts SSH key decryption for the git clone tool.
type SSHKeyProvider interface {
	GetDecryptedKey(userID string, keyID uint) (string, error)
}

// ExtractGitRepoTool clones a git_repo asset, collects file context, and
// generates a resume-oriented analysis report via AI.
type ExtractGitRepoTool struct {
	db           *gorm.DB
	sshProvider  SSHKeyProvider
	aiProvider   AIProvider
	extractModel string
	sizeLimitMB  int
	params       map[string]any
}

// NewExtractGitRepoTool creates a new git repo extraction tool.
func NewExtractGitRepoTool(db *gorm.DB, sshProvider SSHKeyProvider, aiProvider AIProvider, extractModel string, sizeLimitMB int) *ExtractGitRepoTool {
	t := &ExtractGitRepoTool{
		db:           db,
		sshProvider:  sshProvider,
		aiProvider:   aiProvider,
		extractModel: extractModel,
		sizeLimitMB:  sizeLimitMB,
		params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo_asset_id": map[string]any{
					"type":        "integer",
					"description": "git_repo 类型的 Asset ID",
				},
				"user_context": map[string]any{
					"type":        "string",
					"description": "用户的岗位目标和重点（从 interview 技能获取）",
				},
			},
			"required": []string{"repo_asset_id", "user_context"},
		},
	}
	return t
}

func (t *ExtractGitRepoTool) Name() string {
	return "extract_git_repo"
}

func (t *ExtractGitRepoTool) Description() string {
	return "提取 Git 仓库内容并生成简历导向的分析报告。需要提供仓库 Asset ID 和用户岗位上下文。"
}

func (t *ExtractGitRepoTool) Parameters() map[string]any {
	return t.params
}

func (t *ExtractGitRepoTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	assetID, err := getIntParam(params, "repo_asset_id")
	if err != nil {
		return "", fmt.Errorf("repo_asset_id is required: %w", err)
	}
	userContext, _ := params["user_context"].(string)

	var asset models.Asset
	if err := t.db.WithContext(ctx).First(&asset, assetID).Error; err != nil {
		return "", fmt.Errorf("asset not found: %w", err)
	}
	if asset.Type != "git_repo" {
		return "", fmt.Errorf("asset type is %q, not git_repo", asset.Type)
	}
	if asset.URI == nil {
		return "", fmt.Errorf("asset has no URI")
	}
	if asset.KeyID == nil {
		return "", fmt.Errorf("asset has no SSH key associated")
	}

	var project models.Project
	if err := t.db.WithContext(ctx).Select("user_id").First(&project, asset.ProjectID).Error; err != nil {
		return "", fmt.Errorf("project not found: %w", err)
	}

	tempDir, cleanup, err := t.cloneRepo(ctx, *asset.URI, asset.KeyID, project.UserID)
	if err != nil {
		return "", fmt.Errorf("clone failed: %w", err)
	}
	defer cleanup()

	sizeMB, err := dirSizeMB(tempDir)
	if err != nil {
		return "", fmt.Errorf("check size: %w", err)
	}
	if sizeMB > float64(t.sizeLimitMB) {
		return "", fmt.Errorf("仓库大小 %.1fMB 超过限制 %dMB", sizeMB, t.sizeLimitMB)
	}

	report, err := t.exploreRepo(ctx, tempDir, userContext)
	if err != nil {
		return "", fmt.Errorf("explore failed: %w", err)
	}

	analysisLabel := fmt.Sprintf("Git Analysis: %s", *asset.URI)
	analysisAsset := models.Asset{
		ProjectID: asset.ProjectID,
		Type:      "git_analysis",
		Content:   &report,
		Label:     &analysisLabel,
	}
	if err := t.db.WithContext(ctx).Create(&analysisAsset).Error; err != nil {
		return "", fmt.Errorf("save analysis: %w", err)
	}

	t.db.WithContext(ctx).Model(&asset).Update("status", "analyzed")

	result := map[string]any{
		"analysis_asset_id": analysisAsset.ID,
		"repo_url":          *asset.URI,
		"size_mb":           sizeMB,
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

func (t *ExtractGitRepoTool) cloneRepo(ctx context.Context, repoURL string, keyID *uint, userID string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "resume-genius-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { os.RemoveAll(tempDir) }

	repoDir := filepath.Join(tempDir, "repo")
	cloneCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cloneCtx, "git", "clone", "--depth", "1", repoURL, repoDir)
	cmd.Env = os.Environ()

	if t.sshProvider != nil && keyID != nil {
		keyContent, err := t.sshProvider.GetDecryptedKey(userID, *keyID)
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("get SSH key: %w", err)
		}

		keyPath := filepath.Join(tempDir, "id_rsa")
		if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("write SSH key: %w", err)
		}

		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyPath)
		cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+sshCmd)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		cleanup()
		if cloneCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("git clone timed out after 60s")
		}
		return "", nil, fmt.Errorf("git clone failed: %s", stderr.String())
	}

	return repoDir, cleanup, nil
}

// skipDir returns true if the relative path's first component is a well-known
// directory that should be excluded from analysis.
func skipDir(relPath string) bool {
	part, _, _ := strings.Cut(relPath, string(filepath.Separator))
	return part == ".git" || part == "node_modules"
}

// dirSizeMB computes the total size of files under path in megabytes,
// excluding .git and node_modules directories.
func dirSizeMB(path string) (float64, error) {
	var size int64
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(path, p)
		if info.IsDir() && skipDir(rel) {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return float64(size) / (1024 * 1024), err
}

func (t *ExtractGitRepoTool) exploreRepo(ctx context.Context, repoDir, userContext string) (string, error) {
	var contextBuilder strings.Builder

	contextBuilder.WriteString("## 仓库文件树\n")
	filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if err == nil && info.IsDir() {
				rel, _ := filepath.Rel(repoDir, path)
				if skipDir(rel) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		relPath, _ := filepath.Rel(repoDir, path)
		if skipDir(relPath) {
			return nil
		}
		contextBuilder.WriteString(relPath + "\n")
		return nil
	})

	contextBuilder.WriteString("\n## 关键文件\n")
	readKeyFiles(repoDir, &contextBuilder)

	if userContext != "" {
		contextBuilder.WriteString("\n## 用户岗位重点\n")
		contextBuilder.WriteString(userContext)
	}

	systemPrompt := buildGitExploreSystemPrompt()
	userMessage := contextBuilder.String()

	report, err := t.aiProvider.Call(ctx, t.extractModel, systemPrompt, userMessage)
	if err != nil {
		return "", fmt.Errorf("AI analysis failed: %w", err)
	}

	return report, nil
}

func readKeyFiles(repoDir string, sb *strings.Builder) {
	const maxFiles = 10
	const maxBytesPerFile = 5000
	count := 0

	filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || count >= maxFiles {
			if err == nil && info.IsDir() {
				rel, _ := filepath.Rel(repoDir, path)
				if skipDir(rel) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		name := info.Name()
		isKey := name == "README.md" || name == "package.json" || name == "go.mod" ||
			name == "Cargo.toml" || name == "requirements.txt" || name == "Dockerfile" ||
			name == "Makefile" || name == "main.go" || name == "index.ts" || name == "app.py"
		if !isKey {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if len(data) > maxBytesPerFile {
			data = data[:maxBytesPerFile]
		}

		relPath, _ := filepath.Rel(repoDir, path)
		sb.WriteString(fmt.Sprintf("\n### %s\n```\n%s\n```\n", relPath, string(data)))
		count++
		return nil
	})
}

func buildGitExploreSystemPrompt() string {
	return `你是一位技术简历顾问和代码库分析专家。分析提供的仓库上下文，生成简历导向的分析报告。

必须包含：
1. 项目概述（解决什么问题、目标用户、规模）
2. 技术栈全景（语言、框架、数据库、基础设施、前端、AI/ML）
3. 架构要点（3-5 个面试官感兴趣的技术话题）
4. 技术亮点（3-5 个可写进简历的 bullet point，含技术细节）

禁止编造信息。输出中文。`
}
