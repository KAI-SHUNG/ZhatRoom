# ZhatRoom Architecture Redesign

## Context

当前 `hub.go` 是一个 212 行的 god object——混合了客户端注册、消息路由、DB 持久化、snowflake ID、命令处理。`cmd/server/main.go` 的 handshake/validate 逻辑也应下沉到 server 包。TODO 中的命令系统（/users, /nick, /help, /history）、房间系统（/join）、vim 模式都需要清晰的扩展点。

本次重构目标：**拆分职责，为 TODO 功能建立干净的扩展骨架。**

---

## 目标架构

```
cmd/
├── server/main.go          ← 纯入口：config → listen → accept → shutdown
├── client/main.go          ← 纯入口：config → connect → TUI
└── zhatroom/main.go        ← 纯入口：config → admin CLI

internal/
├── config/                  ← 配置（已完成）
├── protocol/                ← 消息协议（保持不变）
│
├── server/
│   ├── server.go            ← Server struct: listen、accept、handshake、shutdown
│   ├── hub.go               ← Hub: 事件循环（register/unregister/message），不含业务逻辑
│   ├── client.go            ← Client: 连接封装、Read/Write goroutine
│   ├── room.go              ← Room: 客户端集合、消息广播范围
│   ├── command.go           ← CommandHandler 注册表 + 内置命令实现
│   ├── message.go           ← 消息处理：ID 生成、持久化、路由到 room
│   └── db.go                ← Storage（保持不变）
│
└── client/tui/
    ├── model.go             ← Model 定义（加 mode 字段）
    ├── update.go            ← Update：按 mode 分发
    ├── keys.go              ← vim 键位绑定定义
    ├── view.go              ← View 渲染（保持不变）
    ├── connector.go         ← Socket 连接（保持不变）
    └── view_test.go         ← 测试（保持不变）
```

---

## 职责划分

### 1. `server/server.go` — 网络层（从 main.go 提取）

**职责**：Unix Socket 监听、连接握手、validate 协议、优雅关闭

```go
type Server struct {
    ln     net.Listener
    hub    *Hub
    cfg    config.ServerConfig
    wg     sync.WaitGroup
}

func NewServer(cfg config.ServerConfig) *Server
func (s *Server) Listen() error              // 监听 socket
func (s *Server) AcceptLoop()                // accept + handshake + validate
func (s *Server) Shutdown()                  // close ln + wg.Wait + hub.Shutdown
```

从 `cmd/server/main.go` 提取：
- socket 文件清理 + `net.Listen`
- accept 循环
- `validate,uid` 协议处理
- `id,nickname` 握手解析
- 信号处理留在 main.go（入口层职责）

### 2. `server/hub.go` — 事件循环（精简）

**职责**：register/unregister/message 事件循环，不含业务逻辑

```go
type Hub struct {
    clients    map[string]*Client
    register   chan *Client
    unregister chan *Client
    broadcast  chan *protocol.Message

    rooms      map[string]*Room
    commands   *CommandRegistry
    store      *Storage
    snowflake  *snowflake.Node
    maxClients int

    mu         sync.RWMutex
}

func NewHub(cfg config.ServerConfig) *Hub
func (h *Hub) Run()                          // select 循环
func (h *Hub) Shutdown()
func (h *Hub) Register(c *Client)
func (h *Hub) Unregister(c *Client)
func (h *Hub) HandleMessage(msg *protocol.Message)
func (h *Hub) Validate(uid string) bool
func (h *Hub) HandleNewConn(conn net.Conn, id, nickname string)
```

从当前 hub.go 移出：
- `/exit` 命令处理 → `command.go`
- 持久化逻辑 → `message.go`
- broadcast 遍历 → `room.go`

### 3. `server/room.go` — 房间抽象（新增）

**职责**：管理房间内的客户端集合，按房间范围广播

```go
type Room struct {
    Name    string
    clients map[string]*Client
    mu      sync.RWMutex
}

func NewRoom(name string) *Room
func (r *Room) Join(c *Client)
func (r *Room) Leave(c *Client)
func (r *Room) Broadcast(msg *protocol.Message)    // 只发给房间内客户端
func (r *Room) Clients() []*Client
func (r *Room) Count() int
```

Hub 持有 `rooms map[string]*Room`，默认创建 `"lobby"` 房间。
Client 持有 `room *Room` 指针，加入房间时更新。

### 4. `server/command.go` — 命令系统（新增）

**职责**：命令注册表 + 内置命令实现

```go
type CommandContext struct {
    Hub     *Hub
    Client  *Client
    Args    []string
    Store   *Storage
}

type CommandHandler func(ctx *CommandContext) *protocol.Message

type CommandRegistry struct {
    handlers map[string]CommandHandler
}

func NewCommandRegistry() *CommandRegistry
func (r *CommandRegistry) Register(name string, handler CommandHandler)
func (r *CommandRegistry) Dispatch(name string, ctx *CommandContext) *protocol.Message
```

内置命令（同包实现）：

| 命令 | 文件 | 说明 |
|------|------|------|
| `/exit` | command.go | 退出聊天室 |
| `/users` | command.go | 列出在线用户 |
| `/nick <name>` | command.go | 修改昵称 |
| `/help` | command.go | 列出可用命令 |
| `/history [n]` | command.go | 加载最近 n 条历史消息 |
| `/join <room>` | command.go | 切换房间 |

### 5. `server/message.go` — 消息处理（从 hub.go 提取）

**职责**：消息 ID 生成、持久化、路由到 room

```go
func (h *Hub) processMessage(msg *protocol.Message) {
    msg.ID = h.snowflake.Generate().String()
    h.store.NewMessage(*msg)

    switch msg.Type {
    case "chat":
        client.room.Broadcast(msg)
    case "command":
        h.commands.Dispatch(msg.Content, ctx)
    }
}
```

### 6. `client/tui/keys.go` — vim 键位（新增）

**职责**：定义 vim 风格键位映射

```go
type Mode int
const (
    ModeInput Mode = iota
    ModeNavigation
)

type KeyMap struct {
    // Navigation mode
    LineUp      key.Binding
    LineDown    key.Binding
    HalfPageUp  key.Binding
    HalfPageDown key.Binding
    GotoTop     key.Binding  // gg
    GotoBottom  key.Binding  // G
    EnterInput  key.Binding  // i

    // Input mode
    Send        key.Binding  // Enter
    EscToNav    key.Binding  // Esc
    Quit        key.Binding  // Ctrl+C
}
```

### 7. `client/tui/update.go` — 按 mode 分发（修改）

当前 Update 直接处理所有按键。改为：

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch m.mode {
        case ModeNavigation:
            return m.updateNavigation(msg)
        case ModeInput:
            return m.updateInput(msg)
        }
    // ... 其他 case 不变
    }
}
```

### 8. `cmd/server/main.go` — 纯入口（精简）

```go
func main() {
    cfg := loadConfig()
    srv := server.NewServer(cfg)
    if err := srv.Listen(); err != nil { ... }
    go srv.AcceptLoop()
    waitForSignal()
    srv.Shutdown()
}
```

约 30 行，只做 config → listen → signal → shutdown。

---

## 实施顺序

### Phase 1: Hub 拆分 + Server 提取

1. **新建 `server/server.go`** — 从 `cmd/server/main.go` 提取 listen/accept/handshake
2. **新建 `server/message.go`** — 从 `hub.go` 提取消息处理
3. **精简 `hub.go`** — 只保留事件循环 + Register/Unregister
4. **精简 `cmd/server/main.go`** — 约 30 行纯入口

### Phase 2: 命令系统

5. **新建 `server/command.go`** — CommandRegistry + 6 个内置命令
6. **hub.go 接入命令** — broadcast case 中 command 类型走 Dispatch

### Phase 3: 房间系统

7. **新建 `server/room.go`** — Room struct + Join/Leave/Broadcast
8. **hub.go 接入房间** — 默认 lobby，/join 切换
9. **client.go 加 room 字段**

### Phase 4: TUI vim 模式

10. **新建 `client/tui/keys.go`** — Mode + KeyMap
11. **修改 `model.go`** — 加 mode 字段
12. **修改 `update.go`** — 按 mode 分发

### Phase 5: 历史消息

13. **`server/db.go`** — 加 `GetMessages(room, limit)` 方法
14. **`/history` 命令** — 调用 DB 查询，返回拼接消息

---

## 文件变更清单

| Action | File | 说明 |
|--------|------|------|
| New | `internal/server/server.go` | Server struct: listen/accept/handshake/shutdown |
| New | `internal/server/message.go` | 消息处理：ID生成、持久化、路由 |
| New | `internal/server/room.go` | Room struct: 客户端集合、按房间广播 |
| New | `internal/server/command.go` | CommandRegistry + 6 个内置命令 |
| New | `internal/client/tui/keys.go` | vim Mode + KeyMap 定义 |
| Rewrite | `internal/server/hub.go` | 精简为纯事件循环 |
| Edit | `internal/server/client.go` | 加 room 字段 |
| Edit | `cmd/server/main.go` | 精简为纯入口 |
| Edit | `internal/client/tui/model.go` | 加 mode 字段 |
| Edit | `internal/client/tui/update.go` | 按 mode 分发按键处理 |
| Edit | `internal/server/db.go` | 加 GetMessages 方法 |

---

## 依赖关系

```
Phase 1 (Hub拆分)
  ├── server/server.go  ← 无依赖，新建
  ├── server/message.go ← 依赖 hub.go 现有逻辑
  └── hub.go 精简       ← 依赖 message.go 完成

Phase 2 (命令系统)
  └── server/command.go ← 依赖 hub.go 稳定

Phase 3 (房间系统)
  └── server/room.go    ← 依赖 hub.go + command.go

Phase 4 (vim 模式)
  └── keys.go + update.go ← 独立，可与 Phase 1-3 并行

Phase 5 (历史消息)
  └── db.go + /history  ← 依赖 Phase 2 command.go
```

---

## 验证

1. `go build ./cmd/...` — 三个 binary 构建通过
2. `go vet ./...` — 无报错
3. `go test ./...` — 现有测试 + 新测试通过
4. 手动测试：启动 server + 2 个 client，验证 /users, /nick, /help, /join, /history
5. 手动测试：vim 模式 i/Esc 切换、gg/G 跳转、j/k 滚动
