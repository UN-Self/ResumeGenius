# ResumeGenius 功能划分

更新时间：2026-04-23

## 1. 功能划分原则

- 项目本质是一个简历编辑产品，AI 和所见即所得编辑器是两条并行的编辑路径
- 功能按业务闭环划分，不按前后端或模型能力横向划分
- HTML 是唯一数据源，所有编辑路径最终都是操作 HTML
- v1 demo 目标是跑通"生成简历 + AI 修改 + 手动编辑 + 导出 PDF"的完整闭环

## 2. 一级功能域

### 2.1 项目管理与资料接入

职责：

- 项目创建与管理
- 文件上传（PDF / DOCX / PNG / JPG）
- Git 仓库 URL 接入
- 补充文本资料录入

产出：

- projects 表记录
- assets 表记录（文件元信息、Git URL、补充文本）

### 2.2 文件解析与 AI 初稿生成

职责：

- 解析 PDF 提取文本块和内嵌图片
- 解析 DOCX 提取段落、表格、样式
- 将提取的文本发送给 AI
- AI 根据简历 HTML 模板骨架 + 用户资料生成完整简历 HTML

产出：

- drafts 表记录（html_content 字段存完整简历 HTML）

### 2.3 AI 对话助手

职责：

- 多轮对话（SSE 流式响应）
- AI 读取当前简历 HTML 作为上下文
- 用户描述需求，AI 返回修改后的完整 HTML
- 用户确认后替换编辑器内容

与 v1 的区别：

- 砍掉意图识别（content_modify / style_modify / mixed）
- 砍掉 PatchEnvelope / PatchOp / SuggestionBuilder
- 砍掉 propose/apply 模式，只有"返回 HTML → 用户确认替换"

### 2.4 可视化编辑器

职责：

- TipTap 所见即所得编辑器
- A4 尺寸编辑画布
- 富文本编辑（加粗、斜体、下划线、字号、颜色、对齐、行距）
- 图片上传与拖拽
- Section 拖拽排序
- 原生 undo/redo
- 自动保存（debounce 2 秒）

与 v1 的区别：

- 从结构化表单编辑器改为 TipTap 所见即所得编辑器
- 砍掉 Patch 映射逻辑（不再需要把用户操作翻译成 PatchOp）
- 编辑和预览是同一个东西

### 2.5 版本管理

职责：

- 每次保存或 AI 修改确认后，自动创建 HTML 快照
- 版本列表（版本号 + 时间 + 标签）
- 回退：将历史快照写回 `drafts.html_content` 并自动创建新快照

产出：

- versions 表记录（html_snapshot 字段存完整 HTML，约 5-10KB）

### 2.6 PDF 导出

职责：

- 前端发送当前 HTML → 后端创建异步导出任务
- chromedp 以 A4 尺寸渲染 HTML → 调用 page.PrintToPDF()
- PDF 文件存储至本地文件系统，客户端通过任务轮询获取下载链接
- 并发控制（同一时间只允许一个导出任务）

产出：

- PDF 文件（返回给前端下载）

## 3. v1 demo 必做范围

### 3.1 必须做

- 新建项目
- 上传文件
- 接入 Git 仓库
- 补充文本资料
- PDF / DOCX 解析
- AI 生成初始简历 HTML
- TipTap 编辑器编辑
- AI 对话修改
- 版本管理（快照、列表、回退）
- PDF 导出（chromedp）

### 3.2 可以简化

- Git 仓库首版只抽 README、项目名、技术栈
- AI 对话首版不需要非常复杂的上下文管理
- PDF 导出首版不做权限控制（后续商业化再加）
- 图片 OCR 首版不做，后续通过云端 API 兜底

### 3.3 明确不做

- LaTeX / TeX Live
- 中间 Patch 协议
- 结构化表单编辑器
- 投递系统
- 草图导入
- VLM 主链路
- 移动端适配

## 4. 推荐的 5 人模块切分

### 模块 A：项目管理与资料接入

职责：

- 项目 CRUD
- 文件上传
- Git 仓库接入
- 补充文本录入

输入输出：

- 输入：用户操作
- 输出：projects 表 + assets 表记录

### 模块 B：文件解析与 AI 初稿生成

职责：

- PDF / DOCX 文件解析
- Git 仓库信息抽取
- AI 生成初始简历 HTML

输入输出：

- 输入：assets 表中的文件路径
- 输出：drafts 表中的 html_content

### 模块 C：AI 对话助手

职责：

- 多轮对话会话管理
- AI 流式响应
- HTML 替换确认

输入输出：

- 输入：用户自然语言 + 当前 HTML + 对话历史
- 输出：AI 回复文本 + 修改后的 HTML

### 模块 D：可视化编辑器

职责：

- TipTap 编辑器集成
- A4 画布
- 工具栏
- 自动保存

输入输出：

- 输入：drafts.html_content
- 输出：编辑后的 HTML（PUT 到 drafts API）

### 模块 E：版本管理与 PDF 导出

职责：

- HTML 快照版本管理
- 版本列表和回退
- chromedp PDF 导出

输入输出：

- 输入：当前 HTML
- 输出：版本记录 + PDF 文件

## 5. 当前最应该先冻结的接口

v2 架构极度简化，需要冻结的接口：

1. 数据库表结构（6 张表）
2. 各模块 API 端点
3. 简历 HTML 模板骨架（AI 生成时使用）
4. AI 消息格式约定（文本 + HTML 分隔符）
5. SSE 流式响应格式
