# ZhatRoom TODO

## 概述

ZhatRoom 是一个基于 SSH 传输的 Go 聊天室。用户通过 `ssh chat@host` 连接，SSH 公钥认证即身份，
`command=` 指令自动拉起客户端。

## 架构方案

```
用户 ssh chat@host
     │
     ▼
authorized_keys:
  command="/opt/zhatroom/entrypoint.sh alice",restrict ssh-rsa AAAA...
     │
     ▼
entrypoint.sh  ──→ 查询 DB（用户是否存在/被封禁）──→ 通过则执行 ./client --usr alice
                        │
                        │ 注册工具: zhatroom user add alice < alice.pub
                        │   ├─ 写入 DB: users 表 + ssh_keys 表
                        │   └─ 追加一行到 authorized_keys
```

| 阶段 | 项目 | 状态 | 说明 |
|------|------|------|------|
| **P0** | TCP 网络传输 | ❌ | 目前只支持 Unix Socket，无法远程连接 |
| **P0** | SSH entrypoint 脚本 | ❌ | 接收用户名、查 DB 验证、拉起客户端 |
| **P0** | 注册 CLI 工具 (`zhatroom user add/list/remove`) | ❌ | 管理 key→user 映射 + authorized_keys |
| **P0** | SSH 公钥表 + DB 验证 | ❌ | ssh_keys 表，entrypoint 查询 |
| **P0** | Bubble Tea TUI 客户端 | ❌ | 见下方 TUI 设计 |
| | | | |
| **P1** | 配置系统接入 main.go | ❌ | config/loader.go 已写好但未调用 |
| **P1** | 心跳检测 + 僵尸连接清理 | ❌ | ping/pong + 超时断开 |
| **P1** | 命令系统扩展 (/users, /nick) | ❌ | 目前只有 /exit |
| **P1** | 协议路由器接入 | ❌ | protocol/router.go 已写好但未用 |
| **P1** | 慢客户端保护 (write_queue) | ❌ | 缓冲发送队列，满了断线 |
| | | | |
| **P2** | 房间/频道系统 | ❌ | Room 模型 + 按房间分发 |
| **P2** | 私信 | ❌ | |
| **P2** | 消息历史查询 (/history) | ❌ | DB 已有数据，缺查询命令 |
| **P2** | 客户端重连 (指数退避) | ❌ | 1s → 2s → 4s → ... → 30s |
| | | | |
| **P3** | 自助注册 (SSH 注册通道) | ❌ | 见方案 B |
| **P3** | Web 自助注册 | ❌ | 贴公钥选用户名 |
| **P3** | 速率限制 | ❌ | 防刷屏 |
| **P3** | 日志系统 | ❌ | 替代 fmt.Printf |
| **P3** | TLS/SSL 支持 | ❌ | config 已定义但未用 |
| **P3** | 指标监控 | ❌ | 在线人数、消息速率 |

---

## Bubble Tea TUI 客户端设计

### 选型：Bubble Tea（charmbracelet/bubbletea）

Bubble Tea 的 Elm 架构天然适合这个场景——客户端需要同时处理 socket 接收和用户输入。

### 依赖

- `github.com/charmbracelet/bubbletea` — 框架
- `github.com/charmbracelet/bubbles` — 内置组件（viewport, textinput）
- `github.com/charmbracelet/lipgloss` — 样式

### 架构

```
cmd/client/main.go
    │
    └── tea.NewProgram(model)
            │
            ├── Model
            │   ├── messages    []protocol.Message  ← 消息历史
            │   ├── viewport    viewport.Model      ← 可滚动消息区
            │   ├── input       textinput.Model     ← 底部固定输入框
            │   ├── connector   *Connector           ← socket 连接管理
            │   └── width, height int                ← 终端尺寸
            │
            ├── Update(msg tea.Msg) → (Model, tea.Cmd)
            │   ├── tea.KeyMsg
            │   │   ├── enter    → 发送消息，清空 input
            │   │   ├── j/k      → viewport.LineDown/LineUp
            │   │   ├── gg       → viewport.GotoTop
            │   │   ├── G        → viewport.GotoBottom
            │   │   ├── pgup/dn  → viewport.HalfViewUp/Down
            │   │   └── ctrl+c   → tea.Quit
            │   ├── incomingMsg   → 追加到 messages，viewport 滚动到底
            │   ├── errMsg        → 显示错误，断开处理
            │   └── tea.WindowSizeMsg → 重新布局
            │
            └── View(model) → string
                    ├── viewport.View()  ← 消息区（自动填充剩余空间）
                    ├── 分隔线
                    └── input.View()     ← 固定在底部
```

### 视觉效果

```
┌─────────────────────────────────┐
│  [Alice]: 大家好                 │  ← viewport（可滚动）
│  [Bob]: 嗨 alice                │
│  [System]: Bob 加入了聊天室       │
│  [Alice]: 今天天气不错            │
│                                 │
│  ─────────────────────────────  │  ← 分隔线
│  > 输入消息...                   │  ← 固定输入框
└─────────────────────────────────┘
```

### 网络 I/O

用 tea.Cmd 封装，不阻塞主循环：

```go
func listenForMessages(conn net.Conn) tea.Cmd {
    return func() tea.Msg {
        line, err := reader.ReadBytes('\n')
        if err != nil {
            return errMsg{err: err}
        }
        msg, _ := protocol.FromJSON(line)
        return incomingMsg{msg: msg}
    }
}
```

### vim 风格导航

| 按键 | 操作 |
|------|------|
| `j` / `↓` | 消息区下滚一行 |
| `k` / `↑` | 消息区上滚一行 |
| `gg` | 滚动到顶部 |
| `G` | 滚动到底部（新消息） |
| `Ctrl+d` / `PgDn` | 下翻半页 |
| `Ctrl+u` / `PgUp` | 上翻半页 |
| 输入模式下 `Esc` | 切到导航模式 |
| 导航模式下 `i` | 切到输入模式 |

### 文件组织

```
internal/
  └── client/
      └── tui/
          ├── model.go      — Model 定义 + Init
          ├── update.go     — Update 函数（所有消息处理）
          ├── view.go       — View 渲染
          └── connector.go  — Socket 连接管理（连、收、发、断）
```

### 开发模式

保留 `--dev` 模式，不走 SSH，直接连本地 Unix Socket：

```bash
# 开发时
./client --usr alice --dev

# SSH 模式（由 entrypoint.sh 调用）
./client --usr alice
```

---

## 使用方法

### 开发启动

```bash
# 1. 启动数据库
cd ZhatRoom && docker compose up -d

# 2. 启动服务器
go run cmd/server/main.go

# 3. 另起终端，启动客户端（可开多个）
go run cmd/client/main.go --usr alice
go run cmd/client/main.go --usr bob
```

### 生产部署（SSH 模式）

```bash
# 1. 管理员注册用户
zhatroom user add alice < alice.pub

# 2. 用户连接
ssh chat@host

# 3. 用户断开
# 输入 /exit 或 Ctrl+C
```

### 命令参考

| 命令 | 说明 |
|------|------|
| `/exit` | 退出聊天室 |
| `/users` | 查看在线用户（待实现） |
| `/nick <name>` | 修改昵称（待实现） |
| `/join <room>` | 切换房间（待实现） |
| `/history` | 查看历史消息（待实现） |

---

## 依赖

```bash
# Bubble Tea
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
```
