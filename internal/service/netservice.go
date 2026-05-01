package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/huaky-tec-fornb/go-net-tool/internal/converter"
	"github.com/huaky-tec-fornb/go-net-tool/internal/model"
	"github.com/huaky-tec-fornb/go-net-tool/internal/network"
)

// NetService bridges the frontend UI and the network layer.
// Its exported methods are callable from the frontend JavaScript.
type NetService struct {
	mu         sync.Mutex
	config     model.ConnectionConfig
	connState  model.ConnectionState
	displayMode string // "text" or "hex"

	tcpClient *network.TCPClient
	tcpServer *network.TCPServer
	udpConn   *network.UDPConn

	msgCh       chan model.Message
	newClientCh chan *network.TCPClientConn

	counters model.ByteCounters
	msgID    uint64

	ctx    context.Context
	cancel context.CancelFunc

	// event emitter (set after service is created)
	app *application.App
}

// SetApp sets the application reference for event emission.
func (s *NetService) SetApp(app *application.App) {
	s.app = app
}

// NewNetService creates a new NetService.
func NewNetService() *NetService {
	return &NetService{
		connState:   model.StateDisconnected,
		displayMode: "text",
		msgCh:       make(chan model.Message, 1000),
		newClientCh: make(chan *network.TCPClientConn, 100),
	}
}

// --- Methods callable from frontend ---

// GetLocalIP returns the device's preferred local IP address
// by dialing a UDP connection to determine the outbound interface.
func (s *NetService) GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "127.0.0.1"
	}
	return host
}

// Connect establishes a connection based on the provided config.
func (s *NetService) Connect(config model.ConnectionConfig) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connState == model.StateConnected || s.connState == model.StateConnecting {
		return "error: 已连接"
	}

	s.config = config
	s.connState = model.StateConnecting
	s.emitState()

	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	switch config.Protocol {
	case model.ProtoTCPClient:
		return s.connectTCPClient(config)
	case model.ProtoTCPServer:
		return s.connectTCPServer(config)
	case model.ProtoUDP:
		return s.connectUDP(config)
	default:
		s.connState = model.StateError
		s.emitState()
		return "error: 未知协议类型: " + string(config.Protocol)
	}
}

// Disconnect tears down the current connection.
func (s *NetService) Disconnect() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.disconnect()
	return "ok"
}

// Send sends text data from the frontend.
func (s *NetService) Send(text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connState != model.StateConnected {
		return "error: 未连接"
	}

	data := []byte(text)
	if len(data) == 0 {
		return "error: 发送内容为空"
	}

	n, err := s.write(data)
	if err != nil {
		return fmt.Sprintf("error: 发送失败: %v", err)
	}

	atomic.AddInt64(&s.counters.PacketsSent, 1)
	atomic.AddInt64(&s.counters.BytesSent, int64(n))
	s.emitStats()
	return fmt.Sprintf("ok: 已发送 %d 字节", n)
}

// SendHex sends hex-encoded data (e.g., "48 65 6c 6c 6f").
func (s *NetService) SendHex(hexStr string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connState != model.StateConnected {
		return "error: 未连接"
	}

	cleaned := strings.ReplaceAll(hexStr, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")

	data, err := hex.DecodeString(cleaned)
	if err != nil {
		return fmt.Sprintf("error: 无效的十六进制: %v", err)
	}
	if len(data) == 0 {
		return "error: 发送内容为空"
	}

	n, err := s.write(data)
	if err != nil {
		return fmt.Sprintf("error: 发送失败: %v", err)
	}

	atomic.AddInt64(&s.counters.PacketsSent, 1)
	atomic.AddInt64(&s.counters.BytesSent, int64(n))
	s.emitStats()
	return fmt.Sprintf("ok: 已发送 %d 字节", n)
}

// SendToClient sends data to a specific client (TCP server mode).
func (s *NetService) SendToClient(clientID string, text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connState != model.StateConnected || s.config.Protocol != model.ProtoTCPServer {
		return "error: 未在服务器模式"
	}

	data := []byte(text)
	n, err := s.tcpServer.SendToClient(clientID, data)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	atomic.AddInt64(&s.counters.PacketsSent, 1)
	atomic.AddInt64(&s.counters.BytesSent, int64(n))
	s.emitStats()
	return fmt.Sprintf("ok: 已发送 %d 字节", n)
}

// SendHexToClient sends hex data to a specific TCP client.
func (s *NetService) SendHexToClient(clientID string, hexStr string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connState != model.StateConnected || s.config.Protocol != model.ProtoTCPServer {
		return "error: 未在服务器模式"
	}

	cleaned := strings.ReplaceAll(hexStr, " ", "")
	data, err := hex.DecodeString(cleaned)
	if err != nil {
		return fmt.Sprintf("error: 无效的十六进制: %v", err)
	}

	n, err := s.tcpServer.SendToClient(clientID, data)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	atomic.AddInt64(&s.counters.PacketsSent, 1)
	atomic.AddInt64(&s.counters.BytesSent, int64(n))
	s.emitStats()
	return fmt.Sprintf("ok: 已发送 %d 字节", n)
}

// DisconnectClient disconnects a specific TCP client.
func (s *NetService) DisconnectClient(clientID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tcpServer == nil {
		return "error: 未在服务器模式"
	}
	err := s.tcpServer.DisconnectClient(clientID)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return "ok"
}

// GetState returns current connection info.
func (s *NetService) GetState() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"connState": string(s.connState),
		"protocol":  string(s.config.Protocol),
		"localIp":   s.config.LocalIP,
		"localPort": s.config.LocalPort,
		"remoteIp":  s.config.RemoteIP,
		"remotePort": s.config.RemotePort,
	}
}

// GetClients returns the list of connected TCP clients.
func (s *NetService) GetClients() []model.ClientInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tcpServer == nil {
		return nil
	}
	return s.tcpServer.GetClients()
}

// GetByteCounters returns current byte statistics.
func (s *NetService) GetByteCounters() model.ByteCounters {
	return model.ByteCounters{
		BytesSent:       atomic.LoadInt64(&s.counters.BytesSent),
		BytesReceived:   atomic.LoadInt64(&s.counters.BytesReceived),
		PacketsSent:     atomic.LoadInt64(&s.counters.PacketsSent),
		PacketsReceived: atomic.LoadInt64(&s.counters.PacketsReceived),
	}
}

// SetDisplayMode switches between "text" and "hex" display.
func (s *NetService) SetDisplayMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if mode == "text" || mode == "hex" {
		s.displayMode = mode
	}
}

// SetRemoteAddr updates the remote IP and port (useful for UDP dynamic target changes).
func (s *NetService) SetRemoteAddr(ip string, port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.RemoteIP = ip
	s.config.RemotePort = port
}

// ClearCounters resets byte/packet counters.
func (s *NetService) ClearCounters() {
	atomic.StoreInt64(&s.counters.BytesSent, 0)
	atomic.StoreInt64(&s.counters.BytesReceived, 0)
	atomic.StoreInt64(&s.counters.PacketsSent, 0)
	atomic.StoreInt64(&s.counters.PacketsReceived, 0)
}

// --- Internal methods ---

func (s *NetService) connectTCPClient(config model.ConnectionConfig) string {
	remoteAddr := fmt.Sprintf("%s:%d", config.RemoteIP, config.RemotePort)
	s.tcpClient = network.NewTCPClient(s.msgCh)

	if err := s.tcpClient.Dial(remoteAddr); err != nil {
		s.connState = model.StateError
		s.emitState()
		return fmt.Sprintf("error: 连接失败: %v", err)
	}

	s.connState = model.StateConnected
	s.emitState()
	go s.eventLoop()
	return "ok: 已连接到 " + remoteAddr
}

func (s *NetService) connectTCPServer(config model.ConnectionConfig) string {
	localAddr := fmt.Sprintf("%s:%d", config.LocalIP, config.LocalPort)
	s.tcpServer = network.NewTCPServer(s.msgCh, s.newClientCh)

	if err := s.tcpServer.Listen(localAddr); err != nil {
		s.connState = model.StateError
		s.emitState()
		return fmt.Sprintf("error: 监听失败: %v", err)
	}

	s.connState = model.StateConnected
	s.emitState()
	go s.eventLoop()
	go s.clientNotifyLoop()
	return "ok: 正在监听 " + localAddr
}

func (s *NetService) connectUDP(config model.ConnectionConfig) string {
	localAddr := fmt.Sprintf("%s:%d", config.LocalIP, config.LocalPort)
	s.udpConn = network.NewUDPConn(s.msgCh)

	if err := s.udpConn.Bind(localAddr); err != nil {
		s.connState = model.StateError
		s.emitState()
		return fmt.Sprintf("error: 绑定失败: %v", err)
	}

	s.connState = model.StateConnected
	s.emitState()
	go s.eventLoop()
	return "ok: 已绑定 " + localAddr
}

func (s *NetService) disconnect() {
	if s.cancel != nil {
		s.cancel()
	}

	if s.tcpClient != nil {
		s.tcpClient.Close()
		s.tcpClient = nil
	}
	if s.tcpServer != nil {
		s.tcpServer.Close()
		s.tcpServer = nil
	}
	if s.udpConn != nil {
		s.udpConn.Close()
		s.udpConn = nil
	}

	s.connState = model.StateDisconnected
	s.emitState()
}

func (s *NetService) write(data []byte) (int, error) {
	switch s.config.Protocol {
	case model.ProtoTCPClient:
		return s.tcpClient.Send(data)
	case model.ProtoTCPServer:
		return 0, fmt.Errorf("服务器模式请选择目标客户端")
	case model.ProtoUDP:
		remoteAddr := fmt.Sprintf("%s:%d", s.config.RemoteIP, s.config.RemotePort)
		if s.config.RemoteIP == "" {
			remoteAddr = ""
		}
		return s.udpConn.SendTo(data, remoteAddr)
	default:
		return 0, fmt.Errorf("未知协议")
	}
}

func (s *NetService) eventLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-s.msgCh:
			if !ok {
				return
			}
			s.processMessage(msg)
		}
	}
}

func (s *NetService) clientNotifyLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.newClientCh:
			s.emitClients()
		}
	}
}

func (s *NetService) processMessage(msg model.Message) {
	if msg.Direction == model.DirSystem {
		isError := msg.Error != ""
		if isError {
			if s.app != nil {
				s.app.Event.Emit("rx-data", map[string]interface{}{
					"text":      msg.Error,
					"timestamp": msg.Timestamp.Format("15:04:05.000"),
					"type":      "system",
					"error":     true,
					"size":      0,
				})
			}
		}
		// Connection events update state
		if strings.Contains(msg.Error, "关闭") || strings.Contains(msg.Error, "断开") {
			s.checkDisconnect(msg)
		}
		return
	}

	// Format the data for display
	var displayText string
	s.mu.Lock()
	mode := s.displayMode
	s.mu.Unlock()

	if mode == "hex" {
		displayText = converter.HexDump(msg.Data, true, 0)
	} else {
		displayText = converter.BytesToEscapedText(msg.Data)
	}

	atomic.AddInt64(&s.counters.PacketsReceived, 1)
	atomic.AddInt64(&s.counters.BytesReceived, int64(msg.Size))

	dir := "接收"
	if msg.Direction == model.DirSent {
		dir = "发送"
	}

	if s.app != nil {
		s.app.Event.Emit("rx-data", map[string]interface{}{
			"text":      displayText,
			"timestamp": msg.Timestamp.Format("15:04:05.000"),
			"direction": dir,
			"srcAddr":   msg.SrcAddr,
			"dstAddr":   msg.DstAddr,
			"size":      msg.Size,
			"clientId":  msg.ClientID,
			"type":      "data",
			"error":     false,
		})
		s.emitStats()
	}
}

func (s *NetService) checkDisconnect(msg model.Message) {
	// If a non-server connection's remote disconnects
	closeMsg := msg.Error

	if strings.Contains(closeMsg, "远程主机") || strings.Contains(closeMsg, "读取错误") {
		if s.config.Protocol != model.ProtoTCPServer {
			s.connState = model.StateDisconnected
			s.emitState()
		}
	}
}

func (s *NetService) emitState() {
	if s.app != nil {
		s.app.Event.Emit("state-change", map[string]interface{}{
			"state": string(s.connState),
		})
	}
}

func (s *NetService) emitStats() {
	if s.app != nil {
		s.app.Event.Emit("stats-update", s.GetByteCounters())
	}
}

func (s *NetService) emitClients() {
	if s.app != nil && s.tcpServer != nil {
		s.app.Event.Emit("clients-update", s.tcpServer.GetClients())
	}
}
