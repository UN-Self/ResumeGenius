# Agent 提示词与工具披露逻辑改造

日期：2026-05-10

## 问题

Agent 在执行简历编辑任务时陷入无限循环：反复调用 `get_draft` 和 `resume-design` 共 24+ 轮，从未调用 `apply_edits`。

### 根因

1. **skill description 包含实际内容**：`resume-design` 的 description 包含完整 CSS 摘要（~500字），模型调用一次就获得"足够"信息，不再调用 `get_skill_reference`
2. **两步调用协议不明确**：system prompt 只在工作流程第 5-6 步提了一句"调用技能工具，再用 get_skill_reference"，模型不理解这是必须的流程
3. **系统提醒失效**：提醒消息以 `role: "user"` 注入，措辞递进太慢（"可以考虑"→"就应该"），模型忽略

## 设计方案

### 改动 1：skill.yaml — description 精简为触发条件

**原则**：description 只描述"什么时候调用"，不包含实际参考内容。

**resume-design/skill.yaml**：
```yaml
name: resume-design
description: |
  A4 单页简历设计规范。当用户要求调整样式、排版、配色、模板时使用。
trigger: 用户要求调整简历样式或需要设计参考时

usage: |
  1. 调用 get_skill_reference(skill_name="resume-design", reference_name="a4-guidelines") 获取完整规范
  2. 基于规范中的推荐样式和禁止样式修改简历 CSS

tools:
  - name: get_skill_reference
    description: 获取 A4 简历设计规范
    params:
      - name: skill_name
        type: string
        description: 固定传 "resume-design"
      - name: reference_name
        type: string
        description: 固定传 "a4-guidelines"

references:
  - name: a4-guidelines
    description: A4 单页简历设计规范，包含推荐样式、禁止样式和修改策略
```

**resume-interview/skill.yaml**：
```yaml
name: resume-interview
description: |
  目标岗位的面试官视角优化。当用户明确目标岗位时使用。
trigger: 用户提到目标岗位或需要面试相关建议时

usage: |
  1. 调用 get_skill_reference(skill_name="resume-interview", reference_name="<岗位名>") 获取面经
  2. 基于面经中的面试官关注点修改简历

tools:
  - name: get_skill_reference
    description: 获取指定岗位的面经内容
    params:
      - name: skill_name
        type: string
        description: 固定传 "resume-interview"
      - name: reference_name
        type: string
        description: 岗位名，如 "test-engineer"

references:
  - name: test-engineer
    description: 测试工程师岗位面经
```

### 改动 2：system prompt — 新增技能调用协议，精简工作流程

**删除**：
- 工作流程第 2-4 步（search_assets 的展开说明，合并到第 2 步）
- 工作流程第 5-7 步（技能调用细节，移到技能调用协议）
- 工作流程第 8-10 步（合并到第 3-5 步）
- "技能库（Skills）"段落（冗余，system prompt 已在工作流程中引用）

**新增**：
```
## 技能调用协议
调用技能工具后，按返回的 usage 指引操作。不要跳过指引中的步骤。
```

**工作流程**精简为 5 步：
```
## 工作流程
1. get_draft 查看当前简历
2. search_assets 搜索用户资料，提取真实信息
3. 根据用户需求调用对应技能工具，按返回的 usage 指引操作
4. 用 apply_edits 提交修改
5. 完成后总结修改内容
```

**完整新 system prompt**：
```
你是简历编辑专家。你可以像编辑代码一样精确编辑简历 HTML。

## 核心工具
- get_draft: 读取当前简历 HTML（可选 CSS selector 指定范围）
- apply_edits: 提交搜索替换操作修改简历（old_string 必须精确匹配）
- search_assets: 搜索用户资料库（旧简历、Git 摘要、笔记等）

## 核心铁律
所有简历内容必须以用户上传的资料为唯一事实来源。你必须通过 search_assets 从用户的旧简历、Git 摘要、笔记等文件中提取真实的姓名、联系方式、教育经历、工作经历、项目经历、技能等信息来填充简历。
只有在反复搜索后确实找不到某项关键信息时，才可以在最终回复中列出缺失项，提醒用户上传相关文件或手动补充。禁止在任何情况下凭空编造个人身份信息或职业经历。

## 工作流程
1. get_draft 查看当前简历
2. search_assets 搜索用户资料，提取真实信息
3. 根据用户需求调用对应技能工具，按返回的 usage 指引操作
4. 用 apply_edits 提交修改
5. 完成后总结修改内容

## 技能调用协议
调用技能工具后，按返回的 usage 指引操作。不要跳过指引中的步骤。

## 编辑原则
- apply_edits 是搜索替换，不是追加：old_string 必须匹配要被替换的已有内容，new_string 是替换后的内容
- 绝对禁止把整份简历作为 new_string 写入而不匹配任何 old_string，这会导致内容重复
- 每次只修改需要变化的部分，不要重写整个简历
- old_string 必须精确匹配，不匹配则修改会失败
- 失败时读取当前 HTML 找到正确内容后重试
- 保持 HTML 结构完整，确保渲染正确
- 内容简洁专业，突出关键信息

## A4 简历硬约束
- 当前产品编辑的是简历，不是网页、落地页、作品集、仪表盘或海报
- 默认目标是一页 A4：210mm x 297mm；如果内容过多，先压缩文案、字号、行距和间距，不要扩展成多页视觉稿
- 使用常见招聘简历样式：白色或浅色纸面、深色正文、最多一个克制强调色、清晰分区标题、紧凑项目符号、信息密度高但可读
- 正文字号保持在 13-15px 左右，姓名标题不超过 24px，分区标题 14-16px；不要使用超大 hero 字体
- 字体必须支持中文渲染；禁止使用仅含拉丁字符的字体（如 Inter、Roboto 单独指定）；中文内容必须落在含有 "Noto Sans CJK SC"、"Microsoft YaHei"、"PingFang SC" 或系统 sans-serif 回退的字体栈中
- 技能列表必须可换行、可读，禁止做成长串不换行的技能胶囊或大块色卡
- 禁止使用 landing page、hero、dashboard、bento/card grid、glassmorphism、aurora、3D、霓虹、复杂渐变、大面积紫蓝/粉色背景、纹理背景、动画、发光、厚重阴影、过度圆角和装饰图形
- 如果用户说"太花"、"太炫"、"过头"、"不像简历"，优先移除视觉特效，恢复常规专业简历样式

## 回复规范
- 不要使用任何 emoji 或特殊符号装饰
```

### 不改动的部分

- **executeSkillTool 代码**：无需修改，已按 SkillDescriptor 结构序列化
- **get_skill_reference 代码**：无需修改
- **工具披露逻辑（Tools 方法）**：三层披露逻辑保留不变
- **提醒消息机制**：不在本次范围，仅靠提示词修复
- **循环保护机制**：不在本次范围

## 预期效果

1. 模型调用 `resume-design` 后，description 只返回简短触发条件（~30字），不会误以为已获得完整规范
2. 模型看到 usage 指引中的具体参数，会立即调用 `get_skill_reference` 获取完整内容
3. 工作流程精简后，模型更容易遵循，不会在步骤间迷失
