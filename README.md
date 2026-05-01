# GoNetTool - 网络调试助手 / Network Debugging Tool

Go 语言编写的跨平台桌面端 TCP/UDP 网络调试工具，支持文本和十六进制数据收发。

A cross-platform desktop TCP/UDP network debugging tool written in Go, with text and hex data send/receive support.

<details open>
<summary><b>🇨🇳 中文</b></summary>

## 功能

### TCP 客户端
- 连接远程 TCP 服务器
- 发送文本或十六进制数据
- 实时显示接收数据（文本/Hex 双模式）
- 毫秒级时间戳

### TCP 服务器
- 监听本地端口，接受多个客户端连接
- 客户端列表管理，选择目标客户端发送数据
- 单独断开指定客户端连接
- 每个客户端独立收发字节统计

### UDP
- 绑定本地端口，接收任意来源数据
- 发送数据到指定 IP/端口，支持广播地址
- 连接后远程地址可动态修改，无需断开重连
- 自动记住上次收到的来源地址，支持快速回复

### 通用功能
- **Hex 显示**：Wireshark 风格（偏移量 + 16字节Hex + ASCII对照）
- **Hex 发送**：输入 `48 65 6c 6c 6f` 自动解码为二进制发送
- **文本显示**：可打印字符正常显示，非打印字符显示为 `.`
- **定时发送**：可配置发送间隔（毫秒级）
- **自动滚动**：接收区自动滚动，可关闭
- **字节统计**：实时统计发送/接收字节数和包数
- **连接状态**：可视化状态指示（绿/红/黄）
- **快捷键**：`Ctrl+Enter` / `Cmd+Enter` 快速发送
- **启动自检**：自动检测设备当前 IP 填入本地 IP

## 界面

```
┌─────────────────────────────────────────────────────────┐
│  协议类型: [TCP客户端 ▼]                                  │
│  本地IP: [192.168.1.100]  本地端口: [7788]               │
│  远程IP: [127.0.0.1]      远程端口: [7788]               │
│  [连接] [断开]    状态: ● 已连接                          │
├─────────────────────────────────────────────────────────┤
│  (TCP服务器模式: 客户端列表下拉框)                         │
├─────────────────────────────────────────────────────────┤
│  接收数据                    [文本 ▼] [清空] [自动滚动 ✓]  │
│  ┌─────────────────────────────────────────────────────┐│
│  │ [16:54:32.001] [接收] Hello World  来源:192.168...  ││
│  │ [16:54:33.456] [发送] Hello World  目标:192.168...  ││
│  └─────────────────────────────────────────────────────┘│
│  发送: 12 字节 | 接收: 24 字节                            │
├─────────────────────────────────────────────────────────┤
│  发送: [________________________] [发送] [Hex发送]       │
│  [□ 定时发送] 间隔: [1000]ms                              │
└─────────────────────────────────────────────────────────┘
```

## 技术栈

| 层 | 技术 |
|---|---|
| 语言 | Go 1.25+ |
| GUI 框架 | [Wails v3](https://v3.wails.io) |
| 前端 | 原生 HTML/CSS/JavaScript（无框架） |
| 构建工具 | Vite + Taskfile |

## 项目结构

```
go-net-tool/
├── main.go                           # 应用入口，Wails 服务注册
├── go.mod / go.sum
├── Taskfile.yml                      # 构建任务定义
├── build/                            # 平台构建配置（macOS/Windows/Linux）
├── frontend/
│   ├── index.html                    # 主界面布局
│   ├── src/main.js                   # 前端逻辑（DOM ↔ Go 服务桥接）
│   ├── public/style.css              # 暗色主题样式
│   ├── dist/                         # Vite 构建产物（嵌入二进制）
│   └── bindings/                     # Wails 自动生成的 Go↔JS 绑定
└── internal/
    ├── model/
    │   ├── message.go                # 消息、方向等基础类型
    │   └── connection.go             # 协议、连接状态、配置类型
    ├── converter/
    │   └── hex.go                    # Wireshark 风格 Hex Dump 格式化
    ├── network/
    │   ├── tcp_client.go             # TCP 客户端：拨号、收发、读循环
    │   ├── tcp_server.go             # TCP 服务器：监听、多客户端管理
    │   └── udp.go                    # UDP：绑定、收发、来源追踪
    └── service/
        └── netservice.go             # Wails 服务层：15个前端可调用方法
```

## 构建

### 前置条件
- Go 1.25+
- Node.js 18+
- Wails v3 CLI

### 安装 Wails v3
```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

### 构建
```bash
export PATH="$HOME/go/bin:$PATH"
wails3 build
```

产物在 `bin/GoNetTool`（macOS arm64 约 7.4MB）。

### 开发模式
```bash
wails3 dev
```
支持热重载（Go 代码和前端代码变更自动重编译）。

## 使用示例

### TCP 客户端
1. 选协议 `TCP 客户端`
2. 远程 IP 填目标地址，端口填目标端口
3. 点 `连接`
4. 输入数据，点 `发送` 或 `Ctrl+Enter`

### TCP 服务器
1. 选协议 `TCP 服务器`
2. 本地端口填监听端口
3. 点 `连接`
4. 客户端连入后从下拉框选择目标，发送数据

### UDP
1. 选协议 `UDP`
2. 本地端口填绑定端口，远程填目标 IP 和端口
3. 点 `连接`
4. 连接后可随时修改远程 IP/端口，发送即生效
5. Hex 模式切换可查看十六进制数据

### Hex 发送
输入十六进制字符串（空格可选），如 `48 65 6c 6c 6f`，点 `Hex发送`。

## 已知问题

- **UDP 广播到 `255.255.255.255` 时**，若本机有多个网卡，接收端可能收到多份。建议使用定向广播地址（如 `192.168.1.255`）或绑定具体网卡 IP 而非 `0.0.0.0`。
- npm 缓存可能出现权限问题，执行 `sudo chown -R $(whoami) ~/.npm` 可修复。

</details>

<details>
<summary><b>🇺🇸 English</b></summary>

## Features

### TCP Client
- Connect to remote TCP servers
- Send text or hex data
- Real-time receive display (text/hex dual mode)
- Millisecond timestamps

### TCP Server
- Listen on a local port, accept multiple client connections
- Client list management with per-client send
- Disconnect individual clients
- Independent byte statistics per client

### UDP
- Bind local port, receive from any source
- Send to specified IP/port, broadcast support
- Remote address editable without reconnecting
- Auto-remembers last received source for quick reply

### General
- **Hex Display**: Wireshark-style (offset + 16-byte hex + ASCII columns)
- **Hex Send**: Input `48 65 6c 6c 6f`, auto-decoded to binary
- **Text Display**: Printable characters shown normally, non-printable as `.`
- **Timed Send**: Configurable interval in milliseconds
- **Auto-scroll**: Toggleable receive area auto-scroll
- **Byte Counters**: Real-time sent/received bytes and packets
- **Status Indicator**: Visual connection state (green/red/yellow)
- **Shortcut**: `Ctrl+Enter` / `Cmd+Enter` to send
- **Startup Detection**: Auto-detects device IP on launch

## UI Layout

```
┌─────────────────────────────────────────────────────────┐
│  Protocol: [TCP Client ▼]                                │
│  Local IP: [192.168.1.100]  Local Port: [7788]          │
│  Remote IP: [127.0.0.1]     Remote Port: [7788]         │
│  [Connect] [Disconnect]    Status: ● Connected           │
├─────────────────────────────────────────────────────────┤
│  (TCP Server: client dropdown hidden otherwise)          │
├─────────────────────────────────────────────────────────┤
│  Received Data              [Text ▼] [Clear] [Auto ✓]    │
│  ┌─────────────────────────────────────────────────────┐│
│  │ [16:54:32.001] [RX] Hello World  src:192.168...    ││
│  │ [16:54:33.456] [TX] Hello World  dst:192.168...    ││
│  └─────────────────────────────────────────────────────┘│
│  Sent: 12 B | Received: 24 B                             │
├─────────────────────────────────────────────────────────┤
│  Send: [________________________] [Send] [Hex Send]     │
│  [□ Auto Send] Interval: [1000]ms                        │
└─────────────────────────────────────────────────────────┘
```

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25+ |
| GUI Framework | [Wails v3](https://v3.wails.io) |
| Frontend | Vanilla HTML/CSS/JavaScript (no framework) |
| Build Tools | Vite + Taskfile |

## Project Structure

```
go-net-tool/
├── main.go                           # App entry point, Wails service registration
├── go.mod / go.sum
├── Taskfile.yml                      # Build task definitions
├── build/                            # Platform build configs (macOS/Windows/Linux)
├── frontend/
│   ├── index.html                    # Main UI layout
│   ├── src/main.js                   # Frontend logic (DOM ↔ Go service bridge)
│   ├── public/style.css              # Dark theme styles
│   ├── dist/                         # Vite build output (embedded in binary)
│   └── bindings/                     # Auto-generated Go↔JS bindings
└── internal/
    ├── model/
    │   ├── message.go                # Message, direction base types
    │   └── connection.go             # Protocol, connection state, config types
    ├── converter/
    │   └── hex.go                    # Wireshark-style hex dump formatting
    ├── network/
    │   ├── tcp_client.go             # TCP client: dial, send/recv, read loop
    │   ├── tcp_server.go             # TCP server: listen, multi-client management
    │   └── udp.go                    # UDP: bind, send/recv, source tracking
    └── service/
        └── netservice.go             # Wails service: 15 frontend-callable methods
```

## Build

### Prerequisites
- Go 1.25+
- Node.js 18+
- Wails v3 CLI

### Install Wails v3
```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

### Build
```bash
export PATH="$HOME/go/bin:$PATH"
wails3 build
```

Output at `bin/GoNetTool` (~7.4 MB on macOS arm64).

### Dev Mode
```bash
wails3 dev
```
Hot reload for both Go and frontend changes.

## Usage

### TCP Client
1. Select `TCP 客户端` protocol
2. Set remote IP and port
3. Click `连接`
4. Type data, click `发送` or `Ctrl+Enter`

### TCP Server
1. Select `TCP 服务器` protocol
2. Set local listening port
3. Click `连接`
4. Select target client from dropdown, send data

### UDP
1. Select `UDP` protocol
2. Set local bind port, remote IP and port
3. Click `连接`
4. Remote IP/port can be changed anytime before sending
5. Toggle hex mode to view hex data

### Hex Send
Input hex string (spaces optional), e.g. `48 65 6c 6c 6f`, click `Hex发送`.

## Known Issues

- **UDP broadcast to `255.255.255.255`**: receivers may get duplicate packets if the host has multiple network interfaces. Use a directed broadcast (e.g. `192.168.1.255`) or bind to a specific interface IP instead of `0.0.0.0`.
- npm cache permission issues: run `sudo chown -R $(whoami) ~/.npm` to fix.

</details>
