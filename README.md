# ZhatRoom —— ssh 聊天室

## Structure

使用 ssh 的公钥，进入后自动启动客户端新进程，使用 unix 本地通信

``` bash
command="./cmd/client/main.go --key_id usr123", <公钥>
```

``` zhatroom
├── README.md
├── bin
├── cmd
│   ├── client
│   │   └── main.go // 客户端入口
│   └── server
│       └── main.go // 服务端入口
├── go.mod
├── go.sum
└── internal
    ├── chat        // 处理聊天信息
    ├── command     // 处理命令
    <!-- ├── config -->
    <!-- │   ├── config.go -->
    <!-- │   └── loader.go -->
    ├── protocol
    │   ├── message.go
    │   └── router.go
    ├── socket
    └── system
```
