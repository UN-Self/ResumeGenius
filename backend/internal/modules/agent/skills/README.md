# Agent Skill 库

本目录按岗位分类存储面经（面试题 + 参考答案 + 面试官关注点）和简历针对性修改建议。

## 目录结构

```
skills/
├── README.md          # 本文件
├── test/              # 测试/QA 岗位
├── tech/              # 技术岗位（待扩展）
├── management/        # 管理岗位（待扩展）
└── creative/          # 创意岗位（待扩展）
```

## 使用方式

System prompt 告知 AI 技能库的存在。AI 在用户明确目标岗位后，调用 `search_skills` 工具按关键词或分类检索匹配的 skill，获取面经内容后应用于简历修改。
