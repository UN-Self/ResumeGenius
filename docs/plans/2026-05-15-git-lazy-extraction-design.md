# Git 仓库延迟提取设计

## 概述

将 git 仓库提取从"上传时立即执行"改为"AI 对话中有目的的延迟执行"。用户上传 git 链接时仅入库，当用户在 AI 对话中表达目标岗位后，AI 自动触发有针对性的仓库分析。

## 动机

当前流程在用户上传 git 链接时立即执行提取，此时 AI 不知道用户的目标岗位，导致分析缺乏针对性。例如用户想投测试岗，AI 却重点分析了后端架构。

## 核心流程

```
用户提交 git URL + 选择/上传 SSH 私钥
        ↓
   [intake 入库]
   SSHKey 表：{alias, encrypted_key, user_id}
   Asset(git_repo) 表：{uri, key_id, status: "pending"}
        ↓
用户在 AI 对话中表达目标岗位
        ↓
   AI 自动 load_skill("resume-interview") → 获得岗位重点
        ↓
   AI 调用 extract_git_repo(repo_asset_id, userContext="岗位重点")
        ↓
   [克隆阶段]
   解密 SSH 私钥 → 写入临时文件
   git clone --depth 1（60s 超时）
   检查仓库大小
   ├─ < 50MB → 继续
   └─ ≥ 50MB → 返回错误，提示仓库过大
        ↓
   [探索阶段]
   派出便宜模型 subagent（GIT_EXTRACT_MODEL 环境变量控制）
   输入：仓库文件树 + 关键源文件 + 岗位重点
   输出：简历导向的分析报告（markdown）
        ↓
   [清理阶段]
   删除临时目录（代码不留存）
   删除临时 SSH 私钥文件
   探索报告存为新 Asset{type: "git_analysis", content: 报告}
   原 git_repo Asset 状态更新为 "analyzed"
        ↓
   AI 后续通过 search_assets 搜索到分析报告
   参考报告 + interview 技能 → 生成针对性简历
```

## 可复用性设计

- SSH 密钥持久化存储，不随 clone 清理，多次提取复用同一密钥
- 每次提取生成新 Asset（git_analysis），同一仓库不同岗位视角各自独立
- 用户换岗位投递时，AI 复用已有密钥重新提取，生成新视角的分析报告

## 数据模型

### 新增表：ssh_keys

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uint | 主键 |
| user_id | string | 用户 ID |
| alias | string | 别名（前端下拉框显示） |
| encrypted_key | text | AES 加密后的 SSH 私钥 |
| created_at | time | 创建时间 |

### Asset 表变更

| 字段 | 变更 | 说明 |
|---|---|---|
| key_id | 新增 | 关联 ssh_keys.id，nullable |
| status | 新增 | pending / analyzed / failed |

### Asset 类型

- `git_repo`：仓库 URL + SSH 密钥引用，状态跟踪
- `git_analysis`：某次提取的分析报告，关联到项目

## API 变更

### 新增：SSH 密钥管理

```
POST   /api/v1/ssh-keys          创建密钥（alias + private_key）
GET    /api/v1/ssh-keys          列出用户的所有密钥（不返回密钥内容）
DELETE /api/v1/ssh-keys/:id      删除密钥
```

### 变更：POST /api/v1/assets/git

请求体新增 `key_id` 字段（选择已有密钥）或 `ssh_key` + `ssh_alias` 字段（上传新密钥）。两者二选一。

### 新增：extract_git_repo 工具

AI 工具，参数：
- `repo_asset_id`：git_repo 类型的 Asset ID
- `userContext`：岗位重点（从 interview 技能获取）

## AI Agent 变动

### 新增工具：extract_git_repo

注册到 `AgentToolExecutor.buildTools()`，实现：
1. 查询 git_repo Asset，获取 URI 和关联的 SSH 密钥
2. 解密密钥，写入临时文件
3. 执行 clone + 大小检测
4. 调用 subagent（GIT_EXTRACT_MODEL）进行探索
5. 清理临时文件和目录
6. 探索报告存为新 Asset

### System Prompt 变更

在 flow_rules 中新增延迟提取引导：
- 当用户表达目标岗位且存在未分析的 git_repo Asset 时，提示 AI 先 load interview 技能，再调用 extract_git_repo

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| GIT_EXTRACT_MODEL | haiku | 探索 subagent 使用的模型 |
| GIT_REPO_SIZE_LIMIT_MB | 50 | 仓库大小限制（MB） |
| SSH_KEY_AES_KEY | — | SSH 私钥加密密钥（必填） |
| AI_API_URL | — | 复用现有，subagent 的 base URL |
| AI_API_KEY | — | 复用现有 |

## 前端变动

### SSH 密钥输入组件

- 下拉选择框：显示已上传密钥的别名
- 选项："上传新密钥"（展开表单：别名输入 + 私钥文本框）
- 提示文案："请使用专用只读密钥，切勿使用生产环境密钥或有写权限的密钥"

### Git 仓库上传流程

1. 用户输入 git URL
2. 选择 SSH 密钥（下拉选择已有 / 上传新密钥）
3. 提交后 Asset 状态显示为"待分析"

## 移除项

- 移除 intake 层对 parsing 模块 git 提取的调用（如有）
- parsing 模块的 `GitRepositoryExtractor` 和 `AIGitExtractor` 逻辑迁移到新工具中
- 保留 parsing 模块的接口定义，供其他解析场景使用

## 安全考虑

- SSH 私钥使用 AES-GCM 加密存储，密钥通过环境变量 `SSH_KEY_AES_KEY` 配置
- clone 时写入临时文件，设置 0600 权限，用完立即删除
- 前端明确提示用户使用专用只读密钥
- API 返回密钥列表时不包含密钥内容，仅返回 id 和 alias
