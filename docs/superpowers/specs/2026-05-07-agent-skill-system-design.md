# Agent 模块 Skill 系统设计

更新时间：2026-05-07

## 1. 背景

AI 对话助手中，用户提出目标岗位（如"我要应聘测试工程师"）后，需要有针对性的面经和简历修改建议。Skill 系统用于按岗位分类存储面经（面试题 + 参考答案 + 面试官关注点），AI 在对话中按需检索并应用到简历修改中。

**核心原则**：渐进式披露。system prompt 仅声明 skill 库可用，不加载具体内容。AI 通过 `search_skills` 工具按需查找匹配的 skill。

## 2. 目录结构

```
backend/internal/modules/agent/skills/
├── README.md              # skill 库总说明
├── test/                  # 测试岗位
│   └── test-resume.yaml   # 测试工程师 skill
```

后续可扩展 `tech/`、`management/`、`creative/` 等分类。

## 3. Skill 文件格式

YAML 格式，遵循结构化 metadata + 自然语言 content 的模式：

```yaml
name: test-resume
description: >-
  面经和简历优化建议，适用于测试工程师岗位
metadata:
  category: test
  keywords:
    - 测试工程师
    - QA
    - 质量保障
  seniority:
    - junior
    - mid
  industries:
    - 互联网
    - 软件
references:
  - title: ISTQB Testing Principles
    source: ISTQB Foundation Level Syllabus
  - title: Google Testing Blog
    source: https://testing.googleblog.com
content: |
  ## 面试经验：测试工程师

  ### 常见面试题
  1. 如何设计一个完整的测试用例？
     ...

  ### 面试官高频关注点
  - 自动化测试框架搭建经验
  ...

  ### 简历针对性要点
  - 项目经历中突出测试框架选型与搭建过程
  ...
```

## 4. 架构

### SkillLoader

新增 `backend/internal/modules/agent/skill_loader.go`

- 使用 Go `embed.FS` 在编译时将 skills/ 目录打包进二进制
- `NewSkillLoader()` 启动时加载所有 .yaml 文件并解析
- `Search(keyword, category, limit)` 运行时检索：
  - keyword 命中 `metadata.keywords` → 返回匹配 skill 完整内容
  - category 匹配 → 返回该目录下全部 skill
  - 空参数 → 返回摘要（name + description 列表）
  - limit 限制返回数量

### Tool 集成

在 `tool_executor.go` 中新增 `search_skills` 工具，与现有 `get_draft`、`apply_edits`、`search_assets` 并列。OpenAI function calling 格式。

### 系统 Prompt

在 `systemPromptV2` 末尾追加一段 skill 使用说明，告知 AI：
1. 存在 skill 库，按岗位分类
2. 使用 `search_skills` 工具查找
3. 使用时机：用户明确目标岗位后

## 5. 数据流

```
用户: "我要应聘测试工程师"
  → AI 调用 search_skills(keyword: "测试工程师")
  → SkillLoader.Search() 匹配 test/test-resume.yaml
  → 工具返回 skill 完整内容（含面经）
  → AI 读面经，理解面试官关注点
  → AI 调用 get_draft 获取当前简历
  → AI 基于面经建议调用 apply_edits 修改简历
```

## 6. 设计决策

| 决策 | 选择 | 理由 |
|---|---|---|
| 存储格式 | YAML | 比 JSON 更适合手写，支持注释 |
| 打包方式 | embed.FS | 零运行时依赖，部署无需额外文件 |
| 检索策略 | keyword 命中 keywords 数组 | 简单精确，避免 NLP 分词复杂度 |
| 渐进策略 | system prompt + tool | 目录不给 AI，按需搜索 |

## 7. 待办

- [ ] 创建 skill 文件
- [ ] 实现 SkillLoader
- [ ] tool_executor.go 集成
- [ ] system prompt 更新
- [ ] routes.go 初始化
- [ ] 测试文件适配
- [ ] 单元测试
