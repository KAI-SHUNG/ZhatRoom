# ZhatRoom — SSH 聊天室

一个基于 SSH 的终端聊天室。朋友通过 `ssh chat@你的服务器` 即可加入，无需安装任何客户端。
SSH 公钥即身份，服务端自动拉起 TUI 客户端，通过 Unix Socket 与服务端通信。

## 功能

- **SSH 一键接入** — 用户只需一条 `ssh` 命令，无需注册、无需安装
- **公钥认证** — SSH 密钥即身份，无需密码
- **Bubble Tea TUI** — 基于 charmbracelet/bubbletea 的终端界面
- **消息对齐** — 自己的消息右对齐，他人消息左对齐，系统消息居中
- **自动用户管理** — 管理员通过 `zhatroom` CLI 添加/删除用户
- **PostgreSQL 持久化** — 用户信息和消息记录存储在数据库中

## 架构

```
ssh chat@server
  │
  ▼
authorized_keys: command="/opt/zhatroom/entrypoint.sh <uid> <username>",no-port-forwarding,...
  │
  ▼
entrypoint.sh
  ├── 查询 PostgreSQL 验证用户身份
  ├── 通过 → exec /opt/zhatroom/bin/client
  └── 拒绝 → "Access denied" 并断开
  │
  ▼
客户端 TUI ── Unix Socket (/tmp/zhatroom.sock) ──► 服务端 Hub ── PostgreSQL
```

## 项目结构

```
ZhatRoom/
├── cmd/
│   ├── client/main.go        # 客户端入口
│   ├── server/main.go        # 服务端入口
│   └── zhatroom/main.go      # 管理员 CLI
├── internal/
│   ├── client/tui/            # Bubble Tea TUI
│   │   ├── model.go           # Model 定义 + Init
│   │   ├── update.go          # Update 消息处理
│   │   ├── view.go            # View 渲染
│   │   ├── connector.go       # Socket 连接管理
│   │   └── view_test.go       # 渲染测试
│   ├── server/
│   │   ├── hub.go             # 消息分发中心
│   │   ├── client.go          # 服务端客户端连接管理
│   │   └── db.go              # 数据库操作
│   ├── protocol/              # 消息协议定义
│   └── config/                # 配置加载
├── scripts/
│   ├── deploy.sh              # 一键部署脚本
│   ├── entrypoint.sh          # SSH 入口脚本
│   └── zhatroom.service       # systemd 服务文件
├── docker-compose.yml         # PostgreSQL 容器
└── bin/                       # 编译产物
```

## 部署流程

### 前置条件

- Linux 服务器（推荐 Ubuntu 22.04+）
- Go 1.21+
- Docker + Docker Compose
- SSH 服务

### 一键部署

```bash
# 克隆项目
git clone git@github.com:KAI-SHUNG/ZhatRoom.git
cd ZhatRoom

# 执行部署脚本（需要 root）
sudo ./scripts/deploy.sh
```

部署脚本会自动完成：
1. 创建 `chat` 系统用户（shell 为 `/bin/sh`，加入 docker 组）
2. 编译 Go 二进制文件
3. 安装到 `/opt/zhatroom/`
4. 启动 PostgreSQL（Docker 容器 `zhat_db`）
5. 安装并启用 systemd 服务
6. 配置 SSH（为 `chat` 用户指定 authorized_keys 路径）

### 验证部署

```bash
# 检查服务状态
sudo systemctl status zhatroom

# 检查数据库容器
docker ps | grep zhat_db
```

## 新用户操作

### 管理员添加用户

```bash
# 添加用户（从标准输入读取公钥）
sudo zhatroom user add <用户名> < ~/.ssh/id_ed25519.pub

# 示例
sudo zhatroom user add alice < /home/alice/.ssh/id_ed25519.pub
```

此命令会：
- 生成随机 UID
- 在 PostgreSQL 中创建用户记录
- 将公钥写入 `/opt/zhatroom/authorized_keys`，绑定 entrypoint

### 用户连接

```bash
ssh chat@你的服务器IP
```

首次连接会提示确认服务器指纹，之后自动进入聊天室 TUI。

### 用户断开

在聊天室中输入 `/exit` 或按 `Ctrl+C`。

### 管理员命令

```bash
# 列出所有用户
sudo zhatroom user list

# 删除用户（同时移除 authorized_keys 中的公钥）
sudo zhatroom user remove <用户名>
```

## 开发模式

不走 SSH，直接本地连接：

```bash
# 1. 启动数据库
docker compose up -d

# 2. 启动服务端
go run cmd/server/main.go

# 3. 另起终端，启动客户端（可开多个）
go run cmd/client/main.go --usr alice
go run cmd/client/main.go --usr bob
```

## TODO

### 近期

- [ ] 完善 vim 操作支持（输入/导航模式切换、`gg`/`G` 跳转、`j`/`k` 滚动）
- [ ] 完善消息回溯（`/history` 命令，从数据库加载历史消息）
- [ ] 完善聊天命令（`/users` 在线用户、`/nick` 修改昵称、`/help` 帮助）
- [ ] 管理员权限系统（区分普通用户和管理员身份）

### 中期

- [ ] 房间/频道系统（`/join <room>` 切换房间）
- [ ] 私信功能
- [ ] 心跳检测 + 僵尸连接清理
- [ ] 客户端断线重连（指数退避）
- [ ] 慢客户端保护（缓冲队列，满了断线）

### 远期

- [ ] 自助注册（SSH 注册通道或 Web 页面）
- [ ] 速率限制（防刷屏）
- [ ] 日志系统（替代 fmt.Printf）
- [ ] 指标监控（在线人数、消息速率）
