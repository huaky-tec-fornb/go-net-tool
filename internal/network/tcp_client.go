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

// TCPClient manages a single outgoing TCP connection.
type TCPClient struct {
	conn   net.Conn
	msgCh  chan<- model.Message
	mu     sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	bytesSent     int64
	bytesReceived int64

	// counter for message IDs
	msgID uint64
}

// NewTCPClient creates a new TCP client that pushes messages to msgCh.
func NewTCPClient(msgCh chan<- model.Message) *TCPClient {
	return &TCPClient{
		msgCh: msgCh,
	}
}

// Dial connects to remoteAddr. Returns error if connection fails.
func (c *TCPClient) Dial(remoteAddr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancel
	c.done = make(chan struct{})

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", remoteAddr)
	if err != nil {
		cancel()
		return fmt.Errorf("dial %s: %w", remoteAddr, err)
	}
	c.conn = conn
	c.msgCh <- model.Message{
		Timestamp: time.Now(),
		Direction: model.DirSystem,
		DstAddr:   remoteAddr,
		Error:     "",
	}

	go c.readLoop()
	return nil
}

// Send writes data to the connection.
func (c *TCPClient) Send(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return 0, fmt.Errorf("未连接")
	}
	n, err := c.conn.Write(data)
	if err != nil {
		return n, err
	}
	c.bytesSent += int64(n)

	c.msgID++
	c.msgCh <- model.Message{
		ID:        c.msgID,
		Timestamp: time.Now(),
		Direction: model.DirSent,
		Data:      data[:n],
		DstAddr:   c.conn.RemoteAddr().String(),
		Size:      n,
	}
	return n, nil
}

// Close closes the connection and stops the read goroutine.
func (c *TCPClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *TCPClient) readLoop() {
	defer close(c.done)

	buf := make([]byte, 4096)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				c.msgCh <- model.Message{
					Timestamp: time.Now(),
					Direction: model.DirSystem,
					Error:     "远程主机关闭了连接",
				}
				return
			}
			// Check if context was cancelled
			select {
			case <-c.ctx.Done():
				return
			default:
			}
			c.msgCh <- model.Message{
				Timestamp: time.Now(),
				Direction: model.DirSystem,
				Error:     fmt.Sprintf("读取错误: %v", err),
			}
			return
		}
		if n > 0 {
			c.bytesReceived += int64(n)
			data := make([]byte, n)
			copy(data, buf[:n])

			c.msgID++
			c.msgCh <- model.Message{
				ID:        c.msgID,
				Timestamp: time.Now(),
				Direction: model.DirReceived,
				Data:      data,
				SrcAddr:   c.conn.RemoteAddr().String(),
				Size:      n,
				ClientID:  "tcp-client",
			}
		}
	}
}
