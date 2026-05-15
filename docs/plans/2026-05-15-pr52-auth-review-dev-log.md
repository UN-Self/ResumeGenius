# PR52 注册认证链路 Review 开发日志

**类型：** Review 开发日志 / 合并前风险记录  
**模块：** auth / workbench / marketing  
**时间：** 2026-05-15  
**相关分支：** `feat/auth-register-email-verification`  
**范围说明：** 本日志只记录 PR52 自身功能链路问题；与 `upstream/dev` 的冲突处理暂不纳入，留到合并收尾阶段统一处理。

---

## 1. 背景

PR52 引入注册认证、邮箱验证码、个人中心、积分/套餐展示，以及营销站和工作台之间的登录态联动。

这条链路的审核重点不是页面能不能打开，而是：

1. 注册后是否必须完成邮箱验证才能登录。
2. 验证码是否真实送达，失败时是否能可靠反馈。
3. 生产环境是否会泄露验证码或留下绕过路径。
4. 用户身份字段是否有数据库层面的唯一性约束。
5. 营销站 `/app` 跳转和登录回跳是否安全。
6. 关键认证路径是否有自动化测试兜底。

当前结论：PR52 的前端生产构建可以通过，但认证核心链路还存在几个会阻止合并的风险点。

---

## 2. 当前 Review 发现

### 2.1 邮箱验证没有真正拦住登录

**位置：**
- `backend/internal/modules/auth/service.go`
- `backend/internal/modules/auth/handler.go`

**问题：**

`Login` 的注释说明未验证邮箱应该返回 `ErrEmailNotVerified`，但实际逻辑只校验账号密码，密码正确后直接返回用户。

结果是：

- 用户注册后即使没有输入验证码，也可以直接登录。
- `/auth/verify-email` 变成 UI 流程上的步骤，而不是后端强约束。
- 上游审核者会认为“邮箱验证注册”这个核心目标没有闭环。

**建议修复：**

- 在密码校验成功后检查：
  - 有邮箱的账号：`email_verified=false` 时拒绝登录。
  - 无邮箱的历史账号：本阶段不做兼容，直接拒绝登录，提示使用邮箱重新注册。
- 增加明确错误，例如 `ErrAccountNeedsEmailRegistration`，不要伪装成“用户名或密码错误”。
- `handler.Login` 增加 `ErrEmailNotVerified` 和无邮箱旧账号错误分支，返回明确错误文案。
- 当前产品尚未正式上线，历史无邮箱账号主要来自开发/测试数据，因此 PR52 阶段可以接受“重注册邮箱账号”策略；如果后续存在需要保留项目/简历数据的真实用户，再单独设计账号迁移或邮箱绑定流程。
- 补测试覆盖：
  - 未验证邮箱 + 正确密码不能登录。
  - 已验证邮箱 + 正确密码可以登录。
  - 无邮箱 legacy 用户登录失败，并提示重新注册邮箱账号。

---

### 2.2 SMTP 失败被吞掉，且验证码会进入生产日志

**位置：**
- `backend/internal/modules/auth/email.go`
- `backend/internal/modules/auth/service.go`

**问题：**

生产模式下，SMTP 连接、认证、发件人、收件人、写入邮件等步骤失败时，`SendVerificationCode` 只是写日志然后 `return nil`。

这会造成两个后果：

1. 后端告诉前端“验证码已发送”，但用户实际收不到邮件。
2. 日志里会出现验证码内容，生产环境存在敏感信息泄露风险。

**建议修复：**

- dev mode 下可以打印验证码，但生产模式不得在日志里输出验证码。
- 生产模式下 SMTP 失败应返回 error，让注册/重发接口给出失败响应。
- SMTP 配置缺失和 SMTP 发送失败要区分：
  - 缺失配置：进入 dev mode 或启动时明确提示。
  - 配置完整但发送失败：接口返回失败，不伪装成功。

### 2.3 开发环境 SMTP 配置说明（非阻塞）

**位置：**
- `.env.example`
- `backend/.env.example`

**结论：**

真实 `.env` 不进仓库，SMTP 账号、密码、发件人等配置本来就应该由开发者或部署环境自行填写。因此 2.3 不作为 PR52 的合并阻塞项，也不进入本轮修复队列。

需要注意的是，`.env.example` 会进入仓库，所以它最多承担“开发者复制后如何进入 dev mode / production mode”的提示职责。当前代码层面已经保证：

- SMTP 配置缺失时进入 dev mode，后端可返回 `dev_code`。
- SMTP 配置完整但发送失败时，接口返回失败，不再伪装发送成功。

后续如需优化，只需要调整示例说明，不影响认证链路正确性。

### 2.4 Email 没有数据库唯一约束

**位置：**
- `backend/internal/shared/models/models.go`
- `backend/internal/modules/auth/service.go`

**问题：**

`User.Email` 目前只有 `gorm:"size:255"`，没有唯一索引。服务层虽然先查询再创建，但这不是强一致保护。

风险：

- 并发注册时可能绕过应用层检查，产生重复 email。
- 邮箱登录时 `Where("email = ?", email).First(&user)` 在重复数据下行为不稳定。
- 账号身份字段没有数据库约束，上游审核通常会卡住。

**建议修复：**

- 给 `email` 增加唯一索引。
- 数据库层面仍允许 `email=NULL`，用于兼容现有数据结构和迁移过程；登录层面不允许无邮箱用户继续使用。
- 注册创建和未验证账号覆盖逻辑用事务包住。
- 对数据库唯一冲突做错误转换，返回 `ErrEmailTaken` 或 `ErrUsernameTaken`。

---

### 2.5 登录 redirect 存在开放跳转风险

**位置：**
- `frontend/workbench/src/pages/LoginPage.tsx`
- `frontend/marketing/src/pages/pricing.astro`
- `frontend/marketing/src/layouts/BaseLayout.astro`

**问题：**

登录页读取 `redirect` 参数后直接执行 `window.location.assign(redirectParam || '/')`，没有限制回跳目标。

风险：

- 构造 `/app/login?redirect=https://example.com` 可在登录成功后跳到外部站点。
- 这类开放跳转在认证入口通常会被审核者认为是安全问题。

**建议修复：**

- 只允许站内路径回跳。
- 推荐规则：
  - 必须以 `/` 开头。
  - 禁止 `//example.com` 这种协议相对 URL。
  - 默认回到 `/app` 或 `/app/`。
- 营销站生成 redirect 时也统一走同一套白名单规则。

---

### 2.6 头像上传读取方式不够稳

**位置：**
- `backend/internal/modules/auth/handler.go`
- `backend/internal/modules/auth/service.go`

**问题：**

头像上传通过一次 `file.Read(buf)` 读取文件内容，没有处理 `Read` 只返回部分数据的情况。虽然对常见 multipart 文件通常能工作，但这不是可靠的流读取方式。

**建议修复：**

- 使用 `http.MaxBytesReader` 或 `io.LimitReader` 限制大小。
- 使用 `io.ReadAll` 读取限定后的内容。
- 先检查 content type / 解码失败路径，再进入压缩保存。

---

### 2.7 关键 auth 链路没有测试覆盖

**位置：**
- `backend/internal/modules/auth`
- `frontend/workbench/tests`

**验证结果：**

- `go test ./internal/modules/auth ./internal/shared/middleware ./internal/shared/models` 可以执行，但 `auth` 包显示 `[no test files]`。
- `frontend/workbench` 和 `frontend/marketing` 的生产构建可以通过。
- `bun test` 当前因 `@/...` alias 解析失败全红，不能给 PR52 的前端登录注册链路提供有效信号。
- `go test ./...` 在本地还受 PostgreSQL、`pdftotext` 等外部依赖影响失败，不能直接作为 PR52 的精确判断。

**建议补测：**

后端 auth service / handler：

- 注册新用户：生成验证码，`email_verified=false`，赠送 100 积分。
- 未验证用户不能登录。
- 验证码错误、过期、成功三条路径。
- 重发验证码会刷新 code 和 expiry。
- 已验证 email 不能重复注册。
- username / email 唯一冲突返回稳定错误。
- dev mode 返回 `dev_code`，production mode 不返回。

前端：

- 注册表单密码确认校验。
- 注册成功进入验证码步骤。
- dev mode 下展示验证码。
- 验证成功跳转登录页。
- 登录 redirect 只允许站内路径。

---

## 3. 本轮修复状态（2026-05-15）

### 已完成

| 问题 | 状态 | 处理结果 |
|------|------|----------|
| 2.1 邮箱验证没有真正拦住登录 | 已修复 | `Login` 现在要求账号必须有邮箱且 `email_verified=true`；无邮箱旧账号直接拒绝登录，并提示使用邮箱重新注册。 |
| 2.2 SMTP 失败被吞掉、验证码进生产日志 | 已修复 | 生产模式 SMTP 连接、认证、收件人、写入等失败会返回 error；生产日志不再输出验证码。dev mode 仍会显示验证码。 |
| 2.4 Email 无数据库唯一约束 | 已修复 | `users.email` 增加 `uniqueIndex`；注册写用户和注册赠送积分改为事务；唯一约束错误会映射为 `ErrEmailTaken` / `ErrUsernameTaken`。 |
| 2.5 登录 redirect 开放跳转风险 | 已修复 | 登录页新增站内路径白名单，只允许 `/...` 且禁止 `//...`、`/\...` 和控制字符；非法 redirect 回落到工作台默认首页。 |
| 2.6 头像上传读取方式不稳 | 已修复 | 上传读取改为 `io.LimitReader` + `io.ReadAll`，避免单次 `Read` 只读到部分内容；继续保留 5MB 限制。 |
| 2.7 关键 auth 链路测试覆盖不足 | 已修复 | 新增 auth service / handler 测试，覆盖未验证邮箱拒绝登录、无邮箱旧账号拒绝登录、注册验证码与积分、重复邮箱/用户名、重发验证码、验证码验证和 dev_code 返回。 |

### 非阻塞

| 问题 | 状态 | 原因 |
|------|------|------|
| 2.3 开发环境 SMTP 配置说明 | 非阻塞 | 真实 `.env` 不进仓库，开发者自行填写；仅保留为 `.env.example` 提示项，不作为合并阻塞。 |

### 仍需注意

- `users.email` 新增唯一索引后，如果目标数据库里已经存在重复的非空 email，迁移会失败；部署前需要先清理重复数据。
- 注册流程选择“先提交数据库事务，再发送邮件”。如果 SMTP 失败，接口会返回失败，但未验证账号和新验证码会保留，用户后续可重试注册或重发验证码。
- `bun test` 的 `@/...` alias 解析问题仍未处理，但 auth 后端核心链路已有独立 Go 测试覆盖。

---

## 4. 后续修复顺序

### Phase 1：合并收尾

合并 `upstream/dev`、处理文档目录移动和其他冲突放到最后。本日志暂不展开冲突处理细节。

---

## 5. 合并前验收标准（当前状态）

- [x] 未验证邮箱不能登录。
- [x] 无邮箱旧账号不能登录，并提示重新注册邮箱账号。
- [x] 验证码发送失败时接口返回失败，前端能展示错误。
- [x] 生产日志不出现验证码。
- [x] 本地 dev mode 能稳定拿到 `dev_code`。（SMTP 配置由开发/部署环境自行填写，2.3 不作为阻塞项）
- [x] `email` 在数据库层面具备唯一性约束。
- [x] 登录回跳无法跳转到外部 URL。
- [x] auth 核心路径有后端测试覆盖。
- [x] workbench / marketing 生产构建通过。

---

## 6. 本轮已执行命令

```bash
gofmt -w backend/internal/modules/auth/service.go backend/internal/modules/auth/handler.go backend/internal/modules/auth/email.go backend/internal/shared/models/models.go
gofmt -w backend/internal/modules/auth/service_test.go backend/internal/modules/auth/handler_test.go
go test ./internal/modules/auth -v
go test ./internal/modules/auth ./internal/shared/middleware ./internal/shared/models
go test ./cmd/server ./internal/modules/auth ./internal/shared/middleware ./internal/shared/models
bun run build # frontend/workbench
bun run build # frontend/marketing
git diff --check
```

补充说明：

- `auth` 包当前没有测试文件，所以 `go test` 只能说明编译可过，不能说明认证链路行为正确。
- `frontend/marketing` 第一次构建在 Astro 输出 `Complete!` 后触发过一次 Bun Windows 运行时 `UV_HANDLE_CLOSING` assertion；复跑后通过。
- 全量后端测试在当前本地环境受 PostgreSQL、`pdftotext` 等依赖影响，不作为 PR52 单独结论。
- `bun test` 当前因路径 alias 解析失败不能提供有效回归信号，本轮未改。
