# 编辑器内容保护设计

## 目标与原则

- **目标**：提高免费用户白嫖门槛，推动付费转化
- **原则**：前端阻吓，不追求绝对防御；不破坏编辑体验；不影响 HTML 数据源（PDF 导出无水印）
- **范围**：仅 A4 编辑器区域（canvas）
- **预留**：接口设计支持后续按付费状态动态开关

## 威胁模型

核心威胁：免费用户通过截图、复制粘贴、浏览器打印等方式获取带排版的简历内容。

## 防御层次

| 层 | 措施 | 防御目标 | 改动文件 |
|---|---|---|---|
| 剪贴板 | `copy` 事件拦截，只写入纯文本到 OS 剪贴板 | Ctrl+C 外部粘贴无格式 | EditorPage.tsx |
| 右键 | `onContextMenu preventDefault` | 右键复制菜单 | A4Canvas.tsx |
| 水印 | React 节点 + MutationObserver 防删除 | 截图包含水印 | A4Canvas.tsx |
| 打印 | `@media print` 隐藏内容 + 付费引导 | Ctrl+P 导出 PDF | editor.css |
| 天然门槛 | HTML 依赖外部 CSS（TipTap 类 + editor.css） | DevTools 复制 HTML 无样式 | 已有 |

## 详细设计

### 1. 水印层（A4Canvas.tsx）

在 A4 纸上方新增一个 React 管理的真实 DOM 节点，绝对定位覆盖。

**关键属性**：
- `pointer-events: none` — 不影响编辑操作
- `position: absolute` + `inset: 0` — 完全覆盖 A4 纸
- `z-index: 5` — 位于内容之上（content=0, zoom toolbar=10）
- `user-select: none` — 水印文本不可选中

**水印文案**（旋转 -30°，重复平铺）：
- 主文案：`ResumeGenius 预览`（大号，半透明）
- 引导文案：`导出无水印 PDF 即可去除水印`（小号）

**防删除机制**：
- React reconciliation：节点被删除后下次渲染自动恢复
- `MutationObserver`：监听父容器 `childList` 变化，检测到水印节点被移除时立即重建

**组件接口**：

```tsx
<WatermarkOverlay visible={true} />
```

预留 `visible` prop，默认 `true`，后续根据用户付费状态传入 `false`。

### 2. 剪贴板纯文本过滤（EditorPage.tsx）

在 `useEditor` 配置的 `editorProps.handleDOMEvents` 中拦截 `copy` 事件。

**逻辑**：
1. 获取选区的纯文本（`view.state.doc.textBetween`）
2. `event.preventDefault()` 阻止浏览器默认的 text/html + text/plain 双格式写入
3. 只手动写入 `text/plain` 到 OS 系统剪贴板
4. 关闭网页后 OS 剪贴板仍保留纯文本，外部粘贴只有无格式文字

**不影响**：编辑器内部复制粘贴走 ProseMirror 事务系统，不经过此拦截。

```typescript
editorProps: {
  handleDOMEvents: {
    copy(view, event) {
      const { from, to } = view.state.selection
      const plainText = view.state.doc.textBetween(from, to, '\n')
      event.preventDefault()
      event.clipboardData.setData('text/plain', plainText)
      return true
    }
  }
}
```

### 3. 右键拦截（A4Canvas.tsx）

在 A4 区域最外层容器上添加：

```tsx
onContextMenu={(e) => e.preventDefault()}
```

仅作用于 A4 区域，不影响工作台其他面板。

### 4. `@media print` 打印拦截（editor.css）

```css
@media print {
  .canvas-area,
  .a4-page {
    display: none !important;
  }
  body::after {
    content: "导出无水印 PDF 即可获得完整简历";
    display: block;
    font-size: 20px;
    text-align: center;
    padding-top: 40vh;
  }
}
```

浏览器 Ctrl+P 打印预览中只显示付费引导，不显示简历内容。

## 不影响的事项

- HTML 数据源不变（数据库存的 HTML 无水印）
- PDF 导出不变（chromedp 渲染纯净 HTML）
- 编辑器内复制粘贴不变（ProseMirror 事务系统）
- AI 生成的 HTML 不变（直接写入编辑器）
- 拖拽操作不变（仅拦截 copy 事件）

## 改动范围

| 文件 | 改动类型 |
|---|---|
| `frontend/workbench/src/components/editor/A4Canvas.tsx` | 新增水印组件 + 右键拦截 + MutationObserver |
| `frontend/workbench/src/styles/editor.css` | 新增 @media print 规则 + 水印样式 |
| `frontend/workbench/src/pages/EditorPage.tsx` | useEditor 中新增 copy 事件拦截 |
