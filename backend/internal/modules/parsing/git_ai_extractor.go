package parsing

// DEPRECATED: Git extraction logic has moved to agent/git_extractor_tool.go.
// These files are retained for reference but should not be used in new code.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxContextChars = 80_000
const maxSourceFileBytes = 25_000
const maxLinesPerSourceFile = 80

type AIGitExtractor struct {
	*GitRepositoryExtractor
}

func NewAIGitExtractor(base *GitRepositoryExtractor) *AIGitExtractor {
	return &AIGitExtractor{GitRepositoryExtractor: base}
}

func (a *AIGitExtractor) Extract(repoURL string, userContext string) (*ParsedContent, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil, ErrAssetURIMissing
	}

	repoDir, cleanup, err := a.prepareRepository(repoURL)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	repoName := inferRepositoryName(repoURL, repoDir)

	readme, _ := a.readReadmePreview(repoDir)
	techStack, _ := a.detectTechStack(repoDir)
	structure, _ := a.summarizeTopLevelStructure(repoDir)

	var ctxBuilder contextBuilder
	ctxBuilder.add("## 仓库信息", fmt.Sprintf("- 名称: %s\n- URL: %s", repoName, repoURL))

	if readme != "" {
		ctxBuilder.add("## README", readme)
	}

	if len(techStack) > 0 {
		ctxBuilder.add("## 技术栈", strings.Join(techStack, ", "))
	}

	if len(structure) > 0 {
		ctxBuilder.add("## 顶级目录结构", strings.Join(structure, "\n"))
	}

	if buildCfg := collectBuildConfigs(repoDir, a.readFile, a.readDir); buildCfg != "" {
		ctxBuilder.add("## 构建和配置文件", buildCfg)
	}

	if cursorRules := collectCursorRules(repoDir, a.readFile, a.readDir); cursorRules != "" {
		ctxBuilder.add("## Cursor/IDE 规则", cursorRules)
	}

	if copilot := collectCopilotInstructions(repoDir, a.readFile); copilot != "" {
		ctxBuilder.add("## Copilot 指令", copilot)
	}

	if sourceFiles := collectKeySourceFiles(repoDir, a.readFile, a.readDir); sourceFiles != "" {
		ctxBuilder.add("## 关键源文件", sourceFiles)
	}

	if userContext != "" {
		ctxBuilder.add("## 用户简历需求", userContext+"\n\n请根据上述用户需求调整分析重点——在技术栈全景中标注与用户需求相关的技术，在技术亮点中优先选择和用户简历方向相关的内容。")
	}

	systemPrompt := buildGitAnalysisSystemPrompt()
	userMessage := ctxBuilder.String()

	ctx, cancel := context.WithTimeout(context.Background(), aiChatTimeout)
	defer cancel()

	markdown, err := aiChat(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGitAIAnalysisFailed, err)
	}

	return &ParsedContent{Text: markdown}, nil
}

// --- context builder ---

type contextBuilder struct {
	sections []string
	total    int
}

func (b *contextBuilder) add(title, body string) {
	entry := fmt.Sprintf("\n%s\n%s\n", title, body)
	if b.total+len(entry) > maxContextChars {
		remaining := maxContextChars - b.total
		if remaining > 200 {
			entry = fmt.Sprintf("\n%s\n%s\n...(truncated)", title, body[:remaining-200])
		} else {
			b.sections = append(b.sections, "\n...(remaining context truncated)")
			b.total = maxContextChars
			return
		}
	}
	b.sections = append(b.sections, entry)
	b.total += len(entry)
}

func (b *contextBuilder) String() string {
	return strings.Join(b.sections, "")
}

// --- context collectors ---

func collectCursorRules(repoDir string, readFile func(string) ([]byte, error), readDir func(string) ([]os.DirEntry, error)) string {
	rulesDir := filepath.Join(repoDir, ".cursor", "rules")
	entries, err := readDir(rulesDir)
	if err != nil {
		return ""
	}

	var sb strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := readFile(filepath.Join(rulesDir, entry.Name()))
		if err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n### %s\n```\n%s\n```\n", entry.Name(), string(content)))
	}
	return sb.String()
}

func collectCopilotInstructions(repoDir string, readFile func(string) ([]byte, error)) string {
	path := filepath.Join(repoDir, ".github", "copilot-instructions.md")
	content, err := readFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func collectBuildConfigs(repoDir string, readFile func(string) ([]byte, error), readDir func(string) ([]os.DirEntry, error)) string {
	var sb strings.Builder

	if pkgJSON, err := readFile(filepath.Join(repoDir, "package.json")); err == nil {
		var pkg struct {
			Name            string            `json:"name"`
			Description     string            `json:"description"`
			Scripts         map[string]string `json:"scripts"`
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if json.Unmarshal(pkgJSON, &pkg) == nil {
			if pkg.Name != "" || pkg.Description != "" {
				sb.WriteString(fmt.Sprintf("\n### package.json\n- name: %s\n- description: %s\n", pkg.Name, pkg.Description))
			}
			if len(pkg.Dependencies) > 0 {
				sb.WriteString("- dependencies:\n")
				for k, v := range pkg.Dependencies {
					sb.WriteString(fmt.Sprintf("  - %s: %s\n", k, v))
				}
			}
			if len(pkg.DevDependencies) > 0 {
				sb.WriteString("- devDependencies:\n")
				for k, v := range pkg.DevDependencies {
					sb.WriteString(fmt.Sprintf("  - %s: %s\n", k, v))
				}
			}
			if len(pkg.Scripts) > 0 {
				sb.WriteString("- scripts:\n")
				keys := make([]string, 0, len(pkg.Scripts))
				for k := range pkg.Scripts {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					sb.WriteString(fmt.Sprintf("  - %s: %s\n", k, pkg.Scripts[k]))
				}
			}
		}
	}

	if goMod, err := readFile(filepath.Join(repoDir, "go.mod")); err == nil {
		sb.WriteString("\n### go.mod\n```\n")
		sb.Write(goMod)
		sb.WriteString("\n```\n")
	}

	if reqTxt, err := readFile(filepath.Join(repoDir, "requirements.txt")); err == nil {
		sb.WriteString("\n### requirements.txt\n```\n")
		sb.Write(reqTxt)
		sb.WriteString("\n```\n")
	}

	if pyproject, err := readFile(filepath.Join(repoDir, "pyproject.toml")); err == nil {
		sb.WriteString("\n### pyproject.toml\n```\n")
		sb.Write(pyproject)
		sb.WriteString("\n```\n")
	}

	for _, name := range []string{"Makefile", "makefile", "GNUmakefile"} {
		if content, err := readFile(filepath.Join(repoDir, name)); err == nil {
			sb.WriteString(fmt.Sprintf("\n### %s\n```\n%s\n```\n", name, string(content)))
			break
		}
	}

	for _, name := range []string{"Taskfile.yml", "Taskfile.yaml", "justfile"} {
		if content, err := readFile(filepath.Join(repoDir, name)); err == nil {
			sb.WriteString(fmt.Sprintf("\n### %s\n```\n%s\n```\n", name, string(content)))
			break
		}
	}

	workflowsDir := filepath.Join(repoDir, ".github", "workflows")
	if entries, err := readDir(workflowsDir); err == nil {
		sb.WriteString("\n### CI Workflows\n")
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
				continue
			}
			content, err := readFile(filepath.Join(workflowsDir, entry.Name()))
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("\n#### %s\n```yaml\n%s\n```\n", entry.Name(), string(content)))
		}
	}

	return sb.String()
}

func collectKeySourceFiles(repoDir string, readFile func(string) ([]byte, error), readDir func(string) ([]os.DirEntry, error)) string {
	var sb strings.Builder
	total := 0

	sourceExts := map[string]bool{
		".go": true, ".py": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".rs": true, ".java": true, ".rb": true, ".php": true, ".vue": true, ".svelte": true,
		".c": true, ".h": true, ".cpp": true, ".hpp": true, ".swift": true, ".kt": true,
	}

	entryPointNames := map[string]bool{
		"main.go": true, "app.py": true, "main.py": true, "server.py": true,
		"index.ts": true, "index.tsx": true, "main.ts": true, "main.tsx": true,
		"App.tsx": true, "app.ts": true, "main.rs": true, "lib.rs": true,
		"index.js": true, "index.jsx": true,
	}

	var walkDir func(dir string, depth int)
	walkDir = func(dir string, depth int) {
		if depth > 3 || total >= maxSourceFileBytes {
			return
		}
		entries, err := readDir(dir)
		if err != nil {
			return
		}

		for _, entry := range entries {
			if total >= maxSourceFileBytes {
				return
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" || name == "target" || name == "build" || name == "dist" {
				continue
			}
			fullPath := filepath.Join(dir, name)

			if entry.IsDir() {
				if depth < 3 {
					walkDir(fullPath, depth+1)
				}
				continue
			}

			ext := filepath.Ext(name)
			if !sourceExts[ext] && !entryPointNames[name] {
				continue
			}

			content, err := readFile(fullPath)
			if err != nil || len(content) == 0 {
				continue
			}

			lines := strings.Split(string(content), "\n")
			if len(lines) > maxLinesPerSourceFile {
				lines = lines[:maxLinesPerSourceFile]
			}
			snippet := strings.Join(lines, "\n")

			relPath, _ := filepath.Rel(repoDir, fullPath)
			header := fmt.Sprintf("\n### %s\n```%s\n%s\n```\n", relPath, strings.TrimPrefix(ext, "."), snippet)

			if total+len(header) > maxSourceFileBytes {
				remaining := maxSourceFileBytes - total
				if remaining > 200 {
					sb.WriteString(fmt.Sprintf("\n### %s\n...(truncated)\n", relPath))
				}
				return
			}
			sb.WriteString(header)
			total += len(header)
		}
	}

	walkDir(repoDir, 1)
	return sb.String()
}

// --- prompt ---

func buildGitAnalysisSystemPrompt() string {
	return `你是一位资深技术简历顾问，同时也是代码库分析专家。你正在分析一个 GitHub 仓库，目的是为简历撰写提供精准的技术素材。

请基于提供的仓库上下文（README、源文件、构建配置），生成一份简历导向的分析报告。你的读者是 AI 简历助手，它需要根据这份报告为候选人写出具体、有说服力的简历 bullet point。

## 必须包含的章节

### 1. 文件开头
"# repository_{仓库名}.md" + 一行项目定位（一句话说清这个项目是干什么的，适合写在简历的项目经历标题下方）

### 2. ## 项目概述
用 3-5 句话描述：
- 项目解决什么问题、目标用户是谁
- 项目规模感知（从文件数量、模块数量推断：是个人小工具、中型应用、还是大型系统）
- 如果有 README，提取其中最核心的功能描述，不要逐字翻译，而是提炼要点

### 3. ## 技术栈全景
从源文件和构建配置中提取完整的技术清单，按类别列出：
- **语言**: 主要编程语言
- **框架**: Web 框架、测试框架、ORM 等（注明具体的框架名称和版本号如果可获取）
- **数据库**: 关系型、NoSQL、缓存等
- **基础设施**: Docker、K8s、CI/CD 工具、消息队列等
- **前端**: 如果有前端，列出 UI 框架、构建工具、状态管理、CSS 方案等
- **AI/ML**: 如果有 AI 相关依赖或调用，单独列出

每一项技术都要能在简历中作为一个关键词出现。不要列无关紧要的工具库。

### 4. ## 架构要点
用 3-5 个要点描述项目的核心架构决策，每个要点应该是"面试官会感兴趣的技术话题"：
- 整体架构风格（单体/微服务/前后端分离/插件化等）
- 关键的技术选型及其理由（如果能从代码中推断）
- 核心业务流或数据流的简化描述（1-2 句话能说清的程度）
- 不要深入模块级的依赖关系，那些对简历无用

### 5. ## 技术亮点
从代码中挖掘 3-5 个可以在简历中展示的技术亮点。每个亮点包含：
- **亮点描述**：用一句话说明做了什么（适合直接改写为简历 bullet）
- **技术细节**：引用的具体技术、模式、工具（作为简历中可展开的细节）

示例：
- 亮点描述：实现基于 Celery + Redis 的异步任务调度系统
- 技术细节：ContextTask 确保 Flask 应用上下文传递，RealtimeStatsCollector 实时收集执行指标

如果没有足够的信息支撑某个亮点，就不要编造。

## 严格禁止
- 编造任何不在上下文中出现的信息
- 列出构建、测试、lint 等开发命令（这是简历素材，不是 README）
- 罗列文件或目录结构
- 通用建议或开发实践
- 重复显而易见的信息

输出语言：中文。每句话都要具体、可被简历引用。`
}
