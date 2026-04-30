# Intake 模块设计文档

更新时间：2026-04-28

## 1. 范围

实现 Intake（项目管理与资料接入）模块的全部功能，包括后端 10 个 API 端点和前端 2 个页面 + 5 个组件。这是整条管线的起点，为后续 parsing/agent/workbench/render 提供数据基础。

## 2. 视觉设计

### 设计语言

Warm Editorial — 暖色编辑风。米白纸张质感，衬线标题，暖灰调，有"纸质简历"的温度感。整体克制、专业，不浮躁。

### 配色方案（Light Mode）

| Token | 色值 | 用途 |
|---|---|---|
| `--bg-page` | `#faf8f5` | 页面背景 |
| `--bg-card` | `#ffffff` | 卡片/弹窗背景 |
| `--text-primary` | `#1a1815` | 标题、正文 |
| `--text-secondary` | `#5c5550` | 辅助说明文字 |
| `--text-muted` | `#9c9590` | placeholder、禁用态 |
| `--accent` | `#c4956a` | 主操作按钮、链接、选中态 |
| `--accent-hover` | `#b5855a` | accent hover 态 |
| `--accent-bg` | `#f0ebe4` | 选中行、标签背景 |
| `--border` | `#e8e4df` | 分割线、输入框边框 |
| `--border-focus` | `#c4956a` | 输入框 focus 态 |
| `--success` | `#0d652d` / `#e6f4ea` | 文字 / 背景 |
| `--error` | `#c5221f` / `#fce8e6` | 文字 / 背景 |
| `--warning` | `#b06000` / `#fef7e0` | 文字 / 背景 |

### 暗色方案（Dark Mode，备用）

备用暗色方案 Quiet Luxury，与 Light 共享同一设计语言，后续启用：

| Token | 色值 | 用途 |
|---|---|---|
| `--bg-page` | `#1c1917` | 页面背景 |
| `--bg-card` | `#292524` | 卡片/弹窗背景 |
| `--text-primary` | `#fafaf9` | 标题、正文 |
| `--text-secondary` | `#a8a29e` | 辅助说明文字 |
| `--text-muted` | `#78716c` | placeholder、禁用态 |
| `--accent` | `#d4a574` | 主操作按钮、链接 |
| `--accent-bg` | `#3a3632` | 选中行、标签背景 |
| `--border` | `#44403c` | 分割线、输入框边框 |

暗色切换通过 `prefers-color-scheme` 检测 + 手动切换按钮。所有颜色通过 CSS 变量管理，切换时只需替换变量值。

### 字体

- 标题/Logo：`Georgia, 'Times New Roman', serif`（浏览器自带，无需加载 web font）
- 正文/UI：`-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif`（系统字体栈）
- 给简历工具一种"文字工坊"的气质

### 圆角

| 元素 | 值 |
|---|---|
| 卡片、弹窗 | 8px |
| 按钮、输入框 | 8px |
| 标签 (Tag) | 4px |

### UI 布局决策

### 项目首页（/）

- **列表视图** + 顶部快捷输入框
- 参考 Linear/GitHub Issues 风格，信息密度高
- 输入框 Enter 或点击按钮即可创建项目，降低操作门槛
- 列表单行显示：标题、资料数量、创建时间、状态标签

### 项目详情页（/projects/:projectId）

- **分离视图**：混合资料列表 + 弹窗操作
- 顶部操作按钮栏：上传文件、接入 Git、添加备注
- 所有类型的资料（文件/Git/文本）混合在一个列表中展示
- 上传/录入通过 Dialog 弹窗完成，主页面保持干净

## 3. 用户隔离（轻量方案）

### 方案

在 `projects` 表增加 `user_id` 字段（VARCHAR 36），不做登录流程。

- **前端**：首次访问时在 `localStorage` 生成一个 UUID 作为匿名用户 ID，每次 API 请求通过 `X-User-ID` header 传递。
- **后端**：所有涉及项目的查询和创建都按 `user_id` 过滤。中间件从 header 提取 `user_id`，注入到 Gin context。
- **数据隔离**：每个浏览器只能看到和操作自己的项目。换浏览器/清缓存会"丢失"数据（可接受，后续升级为真正登录时迁移）。

### 影响

- `Project` model 新增 `UserID` 字段 + 索引
- 新增 `middleware.UserIdentify()` 中间件
- `api-client.ts` 从 `localStorage` 读取并附加 header
- 项目列表/创建/详情/删除全部按 `user_id` 过滤
- 级联删除时也按 `user_id` 校验归属

## 4. 后端架构

### 文件结构

```
backend/internal/modules/intake/
├── routes.go         # 路由注册（10 个端点）
├── handler.go        # HTTP handler（参数校验、调用 service、返回响应）
├── service.go        # 业务逻辑（ProjectService、AssetService）
├── storage.go        # 文件存储（FileStorage 接口 + 本地实现）
├── routes_test.go    # 路由集成测试
├── handler_test.go   # handler 单元测试
└── service_test.go   # service 单元测试
```

### 路由注册

```
POST   /api/v1/projects              创建项目
GET    /api/v1/projects              项目列表
GET    /api/v1/projects/:project_id  项目详情
DELETE /api/v1/projects/:project_id  删除项目（级联删除关联 assets + 文件）

POST   /api/v1/assets/upload         上传文件（multipart）
POST   /api/v1/assets/git            接入 Git 仓库
GET    /api/v1/assets?project_id=X   资产列表
DELETE /api/v1/assets/:asset_id      删除资产

POST   /api/v1/assets/notes          添加补充文本
PUT    /api/v1/assets/notes/:note_id 编辑补充文本
```

### 关键设计决策

1. **FileStorage 接口化** — `storage.go` 定义 `FileStorage` 接口（`Save` / `Delete` / `Exists`），本地文件系统作为默认实现。未来换 S3/OSS 只需新实现，不改 service 层。
2. **Handler 零业务逻辑** — handler 只做参数校验 + 调 service + 返回 response。
3. **级联删除** — `DELETE /projects/{id}` 时，先删关联 assets 记录 + 对应文件，再删 project。在 service 层用事务保证原子性。
4. **上传目录** — 通过环境变量 `UPLOAD_DIR` 配置，默认 `./uploads`。目录结构：`uploads/{project_id}/{uuid}_{filename}`。
5. **文件名去重** — 使用 UUID 前缀避免同名文件覆盖。

### 错误码

| 错误码 | HTTP | 含义 |
|---|---|---|
| 1001 | 400 | 文件格式不支持（.pdf/.docx/.png/.jpg/.jpeg） |
| 1002 | 400 | 文件大小超限（≤ 20MB） |
| 1003 | 400 | Git 仓库 URL 无效 |
| 1004 | 404 | 项目不存在 |
| 1005 | 409 | 资料已存在（重复上传同文件） |
| 1006 | 404 | 资料不存在 |

## 5. 前端架构

### 文件结构

```
frontend/workbench/src/
├── pages/
│   ├── ProjectList.tsx          # 项目首页（列表 + 顶部输入框）
│   └── ProjectDetail.tsx        # 项目详情页（混合资料列表 + 操作按钮）
├── components/
│   ├── ui/                      # shadcn/ui 组件（按需安装）
│   │   ├── button.tsx
│   │   ├── input.tsx
│   │   ├── dialog.tsx
│   │   └── textarea.tsx
│   └── intake/
│       ├── ProjectCard.tsx      # 列表单行
│       ├── AssetList.tsx        # 资料混合列表
│       ├── UploadDialog.tsx     # 上传弹窗（拖拽 + 点击）
│       ├── GitRepoDialog.tsx    # Git 仓库接入弹窗
│       ├── NoteDialog.tsx       # 补充文本弹窗（新建/编辑共用）
│       └── DeleteConfirm.tsx    # 删除二次确认弹窗
└── lib/
    └── api-client.ts            # 已有，无需修改
```

### 路由结构

```
/                       → ProjectList（项目首页）
/projects/:projectId    → ProjectDetail（项目详情页）
/editor/:projectId      → 保留占位，后续 workbench 实现
```

### 关键设计决策

1. **shadcn/ui 按需安装** — 只装 4 个基础组件（button、input、dialog、textarea）。
2. **NoteDialog 复用** — 通过 props（`noteId?`）区分新建/编辑模式。
3. **状态管理** — 不引入 Zustand/Redux，用 React 内置 `useState` + `useEffect`。
4. **文件大小校验** — 前端先校验（≤ 20MB），后端再校验。前端拖拽时显示格式提示。

### 交互流程

- 项目首页：输入框 Enter/按钮 → POST 创建 → 列表刷新
- 项目详情页：点击"上传文件" → UploadDialog → POST multipart → 列表刷新
- 项目详情页：点击 Git/备注 → 对应 Dialog → POST/PUT → 列表刷新
- 删除操作：DeleteConfirm 确认 → DELETE → 列表刷新

## 6. 测试策略

### 后端（TDD）

| 测试文件 | 覆盖范围 |
|---|---|
| `service_test.go` | ProjectService CRUD、AssetService 上传/创建/删除、级联删除、边界校验（文件格式、大小、Git URL 格式） |
| `handler_test.go` | 10 个端点的请求解析、参数校验、错误码映射、响应格式 |
| `routes_test.go` | 路由注册正确性（扩展已有测试） |

- 测试数据库使用 **PostgreSQL**（通过 `docker compose up -d postgres` 启动），与生产环境保持一致。
- 每个测试用例使用独立 transaction，测试结束后 rollback，保证测试间隔离。
- 文件上传测试使用 `t.TempDir()` 隔离。

### 前端

| 测试文件 | 覆盖范围 |
|---|---|
| `ProjectList.test.tsx` | 渲染项目列表、创建项目、空状态 |
| `ProjectDetail.test.tsx` | 渲染资料列表、删除确认 |
| `UploadDialog.test.tsx` | 拖拽上传、格式校验提示 |

- mock `api-client`，不依赖后端。

### 新增依赖

- 后端：`github.com/google/uuid`（文件名去重）
- 前端：`@testing-library/react`、`@testing-library/user-event`（若未安装）

## 7. 不做的事

- 不做真正的登录/注册（v1 用匿名 UUID 隔离，后续升级为 GitHub OAuth 或邮箱登录）
- 不做文件内容解析（parsing 模块的事）
- 不做 AI 初稿生成（parsing 模块的事）
- 不做认证/鉴权（用匿名 UUID 隔离，不做 token 校验）
- 不做分页（v1 项目数量有限，列表全量返回）
- 不做移动端适配（v1 不在范围内）
- 不做 OCR 图片识别（v1 图片仅存储）
