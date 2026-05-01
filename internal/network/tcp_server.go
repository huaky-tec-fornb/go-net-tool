package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/huaky-tec-fornb/go-net-tool/internal/model"
)

// TCPClientConn represents a single accepted TCP client connection.
type TCPClientConn struct {
	ID            string
	Conn          net.Conn
	RemoteAddr    string
	ConnectedAt   time.Time
	BytesSent     int64
	BytesReceived int64
}

// TCPServer listens for and manages multiple TCP client connections.
type TCPServer struct {
	listener    net.Listener
	clients     map[string]*TCPClientConn
	clientsMu   sync.RWMutex
	msgCh       chan<- model.Message
	newClientCh chan *TCPClientConn

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	msgID uint64
	mu    sync.Mutex
}

// NewTCPServer creates a new TCP server.
func NewTCPServer(msgCh chan<- model.Message, newClientCh chan *TCPClientConn) *TCPServer {
	return &TCPServer{
		clients:     make(map[string]*TCPClientConn),
		msgCh:       msgCh,
		newClientCh: newClientCh,
	}
}

// Listen starts listening on addr and accepting connections.
func (s *TCPServer) Listen(addr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	s.done = make(chan struct{})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		cancel()
		return fmt.Errorf("监听 %s: %w", addr, err)
	}
	s.listener = listener

	s.msgCh <- model.Message{
		Timestamp: time.Now(),
		Direction: model.DirSystem,
		Error:     "",
		SrcAddr:   addr,
	}

	go s.acceptLoop()
	return nil
}

// SendToClient sends data to a specific connected client.
func (s *TCPServer) SendToClient(clientID string, data []byte) (int, error) {
	s.clientsMu.RLock()
	client, ok := s.clients[clientID]
	s.clientsMu.RUnlock()

	if !ok {
		return 0, fmt.Errorf("客户端 %s 未找到", clientID)
	}

	n, err := client.Conn.Write(data)
	if err != nil {
		return n, err
	}
	client.BytesSent += int64(n)

	s.mu.Lock()
	s.msgID++
	id := s.msgID
	s.mu.Unlock()

	s.msgCh <- model.Message{
		ID:        id,
		Timestamp: time.Now(),
		Direction: model.DirSent,
		Data:      data[:n],
		DstAddr:   client.RemoteAddr,
		Size:      n,
		ClientID:  clientID,
	}
	return n, nil
}

// DisconnectClient disconnects a specific client.
func (s *TCPServer) DisconnectClient(clientID string) error {
	s.clientsMu.Lock()
	client, ok := s.clients[clientID]
	if ok {
		delete(s.clients, clientID)
	}
	s.clientsMu.Unlock()

	if !ok {
		return fmt.Errorf("客户端 %s 未找到", clientID)
	}

	client.Conn.Close()
	s.msgCh <- model.Message{
		Timestamp: time.Now(),
		Direction: model.DirSystem,
		Error:     fmt.Sprintf("客户端 %s 已断开", client.RemoteAddr),
		SrcAddr:   client.RemoteAddr,
	}
	return nil
}

// GetClients returns the list of currently connected clients.
func (s *TCPServer) GetClients() []model.ClientInfo {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	result := make([]model.ClientInfo, 0, len(s.clients))
	for _, c := range s.clients {
		result = append(result, model.ClientInfo{
			ID:            c.ID,
			RemoteAddr:    c.RemoteAddr,
			ConnectedAt:   c.ConnectedAt.Format("15:04:05"),
			BytesSent:     c.BytesSent,
			BytesReceived: c.BytesReceived,
		})
	}
	return result
}

// Close shuts down the server and all client connections.
func (s *TCPServer) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		s.listener.Close()
	}

	s.clientsMu.Lock()
	for id, client := range s.clients {
		client.Conn.Close()
		delete(s.clients, id)
	}
	s.clientsMu.Unlock()
}

func (s *TCPServer) acceptLoop() {
	defer close(s.done)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(500 * time.Millisecond))
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			continue
		}

		clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
		client := &TCPClientConn{
			ID:          clientID,
			Conn:        conn,
			RemoteAddr:  conn.RemoteAddr().String(),
			ConnectedAt: time.Now(),
		}

		s.clientsMu.Lock()
		s.clients[clientID] = client
		s.clientsMu.Unlock()

		s.msgCh <- model.Message{
			Timestamp: time.Now(),
			Direction: model.DirSystem,
			Error:     fmt.Sprintf("客户端 %s 已连接", client.RemoteAddr),
			SrcAddr:   client.RemoteAddr,
		}

		if s.newClientCh != nil {
			s.newClientCh <- client
		}

		go s.handleClient(client)
	}
}

func (s *TCPServer) handleClient(client *TCPClientConn) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		client.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := client.Conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				s.msgCh <- model.Message{
					Timestamp: time.Now(),
					Direction: model.DirSystem,
					Error:     fmt.Sprintf("客户端 %s 断开连接", client.RemoteAddr),
					SrcAddr:   client.RemoteAddr,
				}
			} else {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				s.msgCh <- model.Message{
					Timestamp: time.Now(),
					Direction: model.DirSystem,
					Error:     fmt.Sprintf("客户端 %s 读取错误: %v", client.RemoteAddr, err),
					SrcAddr:   client.RemoteAddr,
				}
			}

			s.clientsMu.Lock()
			delete(s.clients, client.ID)
			s.clientsMu.Unlock()
			client.Conn.Close()
			return
		}

		if n > 0 {
			client.BytesReceived += int64(n)
			data := make([]byte, n)
			copy(data, buf[:n])

			s.mu.Lock()
			s.msgID++
			id := s.msgID
			s.mu.Unlock()

			s.msgCh <- model.Message{
				ID:        id,
				Timestamp: time.Now(),
				Direction: model.DirReceived,
				Data:      data,
				SrcAddr:   client.RemoteAddr,
				Size:      n,
				ClientID:  client.ID,
			}
		}
	}
}
