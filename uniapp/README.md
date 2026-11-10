# 汤圆记账 · 用户端 uni-app（微信小程序优先）

把现有 `web/` 中的用户端（登录/注册、首页、记账、交易列表、统计、账本设置、帮助）用 uni-app（Vue 3 + TypeScript）重写，**优先发布微信小程序**，同时保留 H5 端能力。平台管理端（`web/src/platform/**`）不迁移。

## 环境要求

- Node.js 22（`nvm use 22`）
- 微信开发者工具（小程序）
- 后端接口：开发与构建统一使用正式环境 `https://tangyuanjizhang.zaixianapp.cn`

## 安装

```bash
cd uniapp
npm install
```

> `typescript` 固定为 `5.4.5`，`vue-tsc` 固定为 `1.8.x`，请勿升级（版本不匹配会导致 `vue-tsc` 启动报错）。

## 微信小程序（优先）

### 开发

```bash
npm run dev:mp-weixin
# 产物：dist/dev/mp-weixin
```

1. 打开微信开发者工具 → 导入项目 → 选择 `uniapp/dist/dev/mp-weixin`。
2. 在 `src/manifest.json` 的 `mp-weixin.appid` 填入小程序 AppID（未填写时构建产物为 `touristappid`，仅可用于预览编译，无法自动化/真机调试）。
3. 开发者工具 → 详情 → 本地设置 → 勾选 **不校验合法域名**（若 AppID 尚未配置正式接口域名）。
4. 开发与构建接口统一指向 `https://tangyuanjizhang.zaixianapp.cn`。如需临时连接本地/测试后端，可在微信开发者工具控制台执行：

```js
// 下次启动生效（写入本地缓存）
wx.setStorageSync('ty_api_base', 'http://192.168.0.1:8080/api')
wx.setStorageSync('ty_upload_base', 'http://192.168.0.1:8080')
```

清除覆盖：`wx.removeStorageSync('ty_api_base')`、`wx.removeStorageSync('ty_upload_base')`。

### 构建 / 发布

```bash
npm run build:mp-weixin
# 产物：dist/build/mp-weixin
```

发布前必须：

- 在 `src/manifest.json` 填入正式 AppID；
- 在微信公众平台配置服务器域名：
  - request 合法域名：`https://tangyuanjizhang.zaixianapp.cn`
  - uploadFile 合法域名：同上（OCR、Excel 导入）
  - downloadFile 合法域名：同上（凭证图片、Excel 导出）
- 在「用户隐私保护指引」中声明：相册/相机（上传票据）、选择聊天文件（Excel 导入）。
- 使用微信开发者工具「上传」→ 公众平台提交审核。

### 微信登录配置

小程序登录需要后端配置与小程序同主体的 AppID / AppSecret（环境变量）：

```bash
WECHAT_MINI_APPID=wx1234567890abcdef
WECHAT_MINI_SECRET=your_app_secret
```

- AppID 必须与 `src/manifest.json` 中 `mp-weixin.appid` 一致，否则 `jscode2session` 会报 `invalid appid`。
- 在微信公众平台「开发管理 - 开发设置」配置服务器 IP 白名单（后端出口 IP）。
- 用户首次微信登录时，`/auth/wechat/login` 返回 `bind_token`，前端进入 `pages/auth/wechat-bind`：
  - 已有账号：输入邮箱 + 密码，关联后直接登录；
  - 无账号：填写用户名 + 账本类型，创建新账号并自动登录。

### 小程序端已适配的能力

| 能力 | 实现 |
|---|---|
| 自定义导航栏 | 状态栏 + 微信胶囊按钮避让（`AppHeader`） |
| 多图选择/拍照 | `uni.chooseImage`，串行压缩 + OCR 队列 |
| 图片压缩 | `uni.compressImage`（H5 用 canvas） |
| SHA-256 去重 | 仅 H5（小程序端跳过，后端字段可选） |
| 图表 | uCharts + `uni.createCanvasContext` |
| Excel 导出 | `uni.downloadFile` + `uni.openDocument` |
| Excel 导入 | `uni.chooseMessageFile`（从聊天记录选文件） |
| 长按 OCR 词条 | `@longpress` + `uni.showActionSheet` |
| 全屏看图 | `uni.previewImage` |
| 微信登录 | `uni.login` 获取 code → 后端 `jscode2session`；首次登录可关联已有邮箱账号或创建新账号 |
| 安全区 | `env(safe-area-inset-bottom)` + 状态栏高度 |

## H5

```bash
# 开发（默认 5275，代理 /api /uploads 到正式环境）
npm run dev:h5

# 临时指定接口地址
API_TARGET=http://localhost:8080 npm run dev:h5

# 构建
npm run build:h5   # dist/build/h5
```

H5 与后端同源部署时接口走 `/api`；开发时由 Vite 代理。H5 支持 Excel 本地文件导入/浏览器下载导出。

## 类型检查与构建验证

```bash
npm run type-check     # vue-tsc --noEmit
npm run build:h5
npm run build:mp-weixin
```

## 目录

```
src/
├── api/            # 唯一网络出口 request.ts + 各业务模块
├── components/     # AppPage/AppHeader/AppTabBar/AppModal/图表/记账/交易/设置组件
├── composables/    # useAuthGuard / useRole / useImageSlots（OCR 队列状态机）
├── pages/          # 10 个页面（pages.json 一次性注册）
├── store/          # auth / tenant / meta（分类/商户/角色缓存）
├── styles/         # 公共类
├── utils/          # format/datetime/ui/image/download/nav/storage
├── config.ts       # API_BASE / UPLOAD_BASE / 常量 / 图片地址
├── manifest.json   # 微信 AppID、网络超时等
└── pages.json      # 页面注册与全局样式
```

## 接口与后端

- 所有请求经 `src/api/request.ts`，自动注入 `Authorization` 与 `X-Tenant-ID`。
- 业务错误（HTTP 200 + `code !== 0`）统一抛 `Error(message)`；HTTP 401 清登录态并回登录页。
- 租户提示词接口 `/llm/prompt`（GET/PUT）已补齐到 `server/cmd/main.go`（此前前端调用 404）。

## 已知限制

- 微信小程序 `chooseMessageFile` 只能从聊天记录选择 Excel；非微信小程序端导入/导出降级为提示。
- `@qiun/ucharts` 在个别机型可能 Canvas 异常，组件内已有「图表暂不可用 + 数据列表」兜底。
- 平台管理端仍使用 React 工程（`web/`）。
- 小程序端的图片压缩质量/尺寸由微信基础库决定，无法像 H5 一样精确控制到 1MB。

## Makefile（仓库根目录）

```bash
make uniapp:dev         # H5 开发（5275）
make uniapp:build       # H5 构建
make uniapp:dev:mp      # 微信小程序开发构建（dist/dev/mp-weixin）
make uniapp:build:mp    # 微信小程序生产构建（dist/build/mp-weixin）
```

以上目标不影响现有 `make qa` / `make prod` 发布流程。小程序发布独立于网站发布。
