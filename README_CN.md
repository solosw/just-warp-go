# Just Warp Go

受 [Warp](https://www.warp.dev/) 启发的桌面终端管理应用。基于 **Wails v2 + Go + Vue 3 + TypeScript** 构建。

管理工作区、运行多 Tab 终端（本地 PTY + SSH）、追踪文件变更——全在漂亮的 GUI 中完成。**无需 Git。**

## 功能

### 工作区管理
- 选择本地文件夹，或通过 SSH 连接远程服务器
- 自动识别 `.gitignore` 规则过滤文件
- 自动跳过二进制文件和无关目录
- 最近打开的工作区历史记录

### 终端
- 多 Tab 终端，支持本地 PTY 和 SSH
- 基于 xterm.js 的终端模拟器
- 从已保存的配置一键创建 SSH 终端
- 每个 Tab 独立工作目录，切换 Tab 保持会话运行

### 文件变更追踪
- 基于 fsnotify 的实时文件监听
- 快照式变更检测，不依赖 Git
- Diff 视图，展示逐行增删统计
- **全部接受** / **全部回退**，批量操作
- **接受单个** / **回退单个**，精细控制

### 远程工作区（SSH + SFTP）
- 连接远程服务器，浏览文件系统
- 基于指纹对比（文件大小 + 修改时间）的变更检测
- 按需读取远程文件 Diff

### 其他
- 启动命令 — 终端启动时自动执行预设的 CLI 命令
- SSH 配置管理 — 保存和复用连接配置
- 文件树浏览，点击打开文件
- 代码预览，语法高亮

## 架构

```
Wails 桌面应用
├── Go 后端
│   ├── app.go            — 绑定的方法（工作区、终端、快照、SSH）
│   ├── scanner/          — 递归文件扫描 + .gitignore + 二进制检测
│   ├── watcher/          — fsnotify 封装，带防抖
│   ├── snapshot/         — SHA-256 快照引擎 (.warp-snapshots/)
│   ├── terminal/         — PTY & SSH 会话管理
│   └── config/           — JSON 持久化存储（工作区、SSH、命令）
├── Vue 3 前端
│   ├── WorkspaceBar      — 工作区路径 + 选择按钮 + 历史
│   ├── TerminalPanel     — Tab 栏 + xterm.js 实例
│   ├── FileChangesPanel  — 变更列表 + 接受/回退按钮
│   ├── FileTreePanel     — 工作区文件浏览器
│   ├── FilePreviewPanel  — 文件内容 + Diff 视图
│   └── DiffView          — 并排 Diff 展示
```

## 前置条件

- [Go](https://go.dev/) 1.25+
- [Node.js](https://nodejs.org/) 18+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)

## 快速开始

```bash
# 克隆
git clone https://github.com/solosw/just-warp-go.git
cd just-warp-go

# 安装前端依赖
cd frontend && npm install && cd ..

# 开发模式运行
wails dev

# 构建生产版本
wails build
```

开发服务器启动在 `http://localhost:34115`，支持热重载。

## 技术栈

| 层面 | 选型 |
|------|------|
| 框架 | Wails v2 |
| 后端 | Go |
| 前端 | Vue 3 + TypeScript + Vite |
| 终端 | xterm.js + conpty (Windows) |
| 状态管理 | Pinia |
| 文件监听 | fsnotify |
| 快照存储 | SHA-256 + 文件系统 |

## 开源协议

MIT
