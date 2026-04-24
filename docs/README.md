# ResumeGenius Docs

更新时间：2026-04-23

## 文档体系

文档分为两层：

- **共享规范层**（`01-product/`）：所有模块必读，定义技术选型、功能划分、UI 风格和 API 规约
- **模块契约层**（`modules/`）：5 份模块契约，每份对应一个功能模块，独立可开发、独立可测试

## 目录结构

```text
docs/
  README.md
  prd_v2.md                        # v2 产品需求文档
  prd_v1.md                        # 已废弃（v1 LaTeX 架构）
  01-product/                      # 共享规范
    product-logic-diagrams.md      # 功能关系图 + 用户流程图
    functional-breakdown.md        # 功能划分 + 5 人模块切分
    tech-stack.md                  # 技术栈选型
    ui-design-system.md            # UI 风格规范
    api-conventions.md             # 统一 API 规约
    dev-work-breakdown.md          # 开发工作总览
  02-data-models/                  # 数据模型与 Mock
    core-data-model.md             # 数据库表结构（v2 极简 6 表）
    mock-fixtures.md               # Mock 数据策略 + fixture 示例
  modules/                         # 5 个模块契约
    a-intake/                      # A 资料接入
    b-parsing/                     # B 解析初稿
    c-agent/                       # C AI 对话
    d-workbench/                   # D 可视化编辑
    e-render/                      # E 版本导出
  superpowers/specs/               # 架构设计规格
    2026-04-23-architecture-v2-design.md   # v2 架构设计（已批准）
    2026-04-24-v2-doc-consistency-fix-design.md  # 文档一致性修复设计
```

## 架构总览

```
[A 资料接入] → 文件/资料 → [B 解析初稿] → HTML 初稿
                                       │
                              ┌─────────┴─────────┐
                              ▼                   ▼
                        [C AI 对话]          [D 可视化编辑]
                        AI 返回 HTML          直接编辑 HTML
                              │                   │
                              └─────────┬─────────┘
                                        ▼
                              [E 版本管理 + PDF 导出]
                                HTML 快照 / chromedp
```

**核心原则**：HTML 是唯一数据源，零中间层。

| 模块 | 职责 | 契约文档 |
|---|---|---|
| A 资料接入 | 项目 CRUD、文件上传、Git 接入、补充文本 | [contract.md](./modules/a-intake/contract.md) |
| B 解析初稿 | 解析文件提取文本 → AI 生成简历 HTML | [contract.md](./modules/b-parsing/contract.md) |
| C AI 对话 | 多轮 SSE 对话，AI 返回修改后的 HTML | [contract.md](./modules/c-agent/contract.md) |
| D 可视化编辑 | TipTap 所见即所得编辑，A4 画布 | [contract.md](./modules/d-workbench/contract.md) |
| E 版本导出 | HTML 快照版本管理 + chromedp PDF 导出 | [contract.md](./modules/e-render/contract.md) |

## 共享规范入口

- 产品逻辑图：[product-logic-diagrams.md](./01-product/product-logic-diagrams.md)
- 功能划分：[functional-breakdown.md](./01-product/functional-breakdown.md)
- 技术栈：[tech-stack.md](./01-product/tech-stack.md)
- UI 风格：[ui-design-system.md](./01-product/ui-design-system.md)
- API 规约：[api-conventions.md](./01-product/api-conventions.md)
- 数据结构：[core-data-model.md](./02-data-models/core-data-model.md)
- Mock 策略：[mock-fixtures.md](./02-data-models/mock-fixtures.md)
- 开发工作总览：[dev-work-breakdown.md](./01-product/dev-work-breakdown.md)
- v2 架构设计：[architecture-v2-design.md](./superpowers/specs/2026-04-23-architecture-v2-design.md)

## 开发方式

每个模块的开发者：

1. 先读共享规范层（01-product），理解全局约束
2. 读 [core-data-model.md](./02-data-models/core-data-model.md) 了解数据库表结构
3. 读自己模块的 contract.md + work-breakdown.md
4. 用 mock 数据替代上下游，独立开发测试
5. 不需要等上下游模块完成，只要接口对齐即可

## 备注

- `prd_v2.md` 是 v2 的产品需求文档
- `superpowers/specs/` 目录下存放架构设计规格
- v1 的 6 层数据结构和 Patch 协议已废弃，详见 v2 架构设计文档 §12 和 §13
