# Agent Skill 渐进式披露系统设计

更新时间：2026-05-08

## 1. 问题

PR #42 实现了 skill 系统，但存在两个问题：

1. **不是真正的渐进式披露**：`search_skills` 工具要么不调，要么一步返回全部面经内容（95 行 YAML）。没有中间层级让 AI 先了解"有什么"再决定"拿什么"。
2. **System prompt 前缀不稳定**：如果动态修改 system prompt 注入 skill 内容，会破坏整个会话的 prompt cache 命中率。

## 2. 设计目标

- 三层渐进式披露：索引 → 描述 → 内容，逐层加载
- System prompt 在整个会话中保持不变，确保缓存命中
- Skill 数量少（个位数）时，索引直接放 system prompt，不走工具调用
- AI 必须先加载 skill 描述文档，才能使用其二级工具和 reference

## 3. 三层架构

### Layer 1 — Skill 索引（System Prompt，固定不变）

索引直接写在 system prompt 中，整个会话期间不变化。只暴露 skill 名称和触发条件，**不暴露二级工具和 reference**。

```
## 技能库（Skills）
- resume-interview: 当用户明确了目标岗位（如"测试工程师"、"前端开发"）时使用
```

为什么不暴露二级工具和 reference：
- 防止 AI 跳过描述文档直接调用工具，导致误用
- 描述文档是"使用手册"，必须先读才能正确操作

### Layer 2 — Skill 描述文档（AI 决定使用时加载）

当 AI 根据索引判断需要某个 skill 时，调用 `load_skill("resume-interview")` 获取描述文档。

描述文档包含：
- skill 的用途和使用场景
- 可用的二级工具及正确用法
- reference 列表及各自适用场景

```yaml
name: resume-interview
description: 根据目标岗位的面试官视角优化简历
trigger: 用户提到目标岗位或需要面试相关建议时

usage: |
  1. 确认用户的目标岗位
  2. 使用 get_skill_reference 获取该岗位的面经
  3. 基于面经中的面试官关注点修改简历

tools:
  - name: get_skill_reference
    description: 获取指定岗位的面经内容（面试题、参考答案、面试官关注点、简历修改建议）
    params:
      - name: reference_name
        type: string
        description: 岗位名，如 "test-engineer"、"frontend-developer"
    usage: 调用后会返回该岗位的完整面经，内容较长，确保已确认用户岗位后再调用

references:
  - name: test-engineer
    description: 测试工程师岗位面经，含 18 道面试题及面试官建议
  - name: frontend-developer
    description: 前端开发岗位面经（待补充）
  - name: product-manager
    description: 产品经理岗位面经（待补充）
```

### Layer 3 — Reference 内容（通过二级工具按需加载）

AI 读完描述文档后，根据用户的具体岗位调用 `get_skill_reference("test-engineer")` 拉取对应面经。

reference 文件按岗位独立存放，内容是纯粹的数据（面试题、答案、建议），不含使用说明：

```yaml
# skills/references/test-engineer.yaml
name: test-engineer
content: |
  ## 面试经验：测试工程师

  ### 常见面试题
  1. **黑盒测试有哪些方法？**
     回答要点：...
  ...（完整面经内容）
```

## 4. 数据流

```
用户: "我要应聘测试工程师"

  → AI 读 system prompt 索引，判断命中 resume-interview
  → AI 调用 load_skill("resume-interview")
  → 返回描述文档（usage + tools + references 列表）
  → AI 读描述文档，知道该用 get_skill_reference 工具
  → AI 调用 get_skill_reference("test-engineer")
  → 返回测试工程师面经全文
  → AI 基于面经调用 get_draft + apply_edits 修改简历

用户: "帮我改改简历"

  → AI 读索引，判断不需要 resume-interview
  → 直接调用 get_draft + apply_edits
  → 零额外 token 消耗
```

## 5. 目录结构

```
backend/internal/modules/agent/skills/
├── resume-interview/                # 一个 skill 一个目录
│   ├── skill.yaml                   # Layer 2: 描述文档
│   └── references/                  # Layer 3: reference 内容
│       ├── test-engineer.yaml
│       ├── frontend-developer.yaml  # 待补充
│       └── product-manager.yaml     # 待补充
```

每个 skill 一个独立目录，描述文档和 reference 分开存放。新增岗位只需加一个 reference YAML，不用改代码。

## 6. 工具设计

### load_skill

| 字段 | 值 |
|---|---|
| name | `load_skill` |
| description | 加载指定技能的描述文档，了解技能的用途、可用工具和参考资源 |
| params | `skill_name: string`（必填） |
| required | `["skill_name"]` |
| 返回 | skill 描述文档完整 YAML 内容 |

### get_skill_reference

| 字段 | 值 |
|---|---|
| name | `get_skill_reference` |
| description | 获取技能库中指定岗位的面经内容。必须在 load_skill 之后使用 |
| params | `reference_name: string`（必填） |
| required | `["reference_name"]` |
| 返回 | reference 完整内容 |

## 7. 设计决策

| 决策 | 选择 | 理由 |
|---|---|---|
| 索引位置 | System Prompt 固定文本 | Skill 数量少，直接写入即可；避免 system prompt 变化破坏缓存 |
| 索引不暴露工具/引用 | 只写名称和触发条件 | 防止 AI 跳过描述文档直接操作，避免误用 |
| 描述与内容分离 | 两个独立文件 | 描述是"使用手册"，内容是"数据"，职责不同 |
| reference 按岗位独立 | 每个岗位一个 YAML | 新增岗位只加文件不改代码，按需加载省 token |
| 二级工具统一接口 | 一个 `get_skill_reference` 而非每个 skill 独立工具 | 工具通用，内容通过参数区分；skill 增多不用注册新工具 |
| embed.FS 打包 | 编译时嵌入二进制 | 零运行时依赖，部署无需额外文件 |

## 8. 与 PR #42 现有实现的差异

| 维度 | PR #42 | 新设计 |
|---|---|---|
| 披露层级 | 1.5 层（system prompt → 全量内容） | 3 层（索引 → 描述 → 内容） |
| 搜索工具 | `search_skills` 一个工具干所有事 | `load_skill` + `get_skill_reference` 职责分离 |
| 文件结构 | 一个 YAML 混合描述和内容 | 描述文档和 reference 分开存放 |
| System prompt | 提及工具名 | 只提及 skill 名称，不提工具 |
| 缓存友好 | 无特殊考虑 | System prompt 全会话固定 |

## 9. 后续扩展

- 新增岗位：在 `references/` 下加 YAML 即可，不改代码
- 新增 skill 类型（如"英语简历优化"）：在 skills/ 下新建目录，system prompt 加一行索引
- reference 内容增多后可加 `get_skill_reference` 的分页或章节参数（如只取"简历修改建议"部分）