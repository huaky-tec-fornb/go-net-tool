package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/huaky-tec-fornb/go-net-tool/internal/model"
)

// UDPConn manages a UDP socket for sending and receiving datagrams.
type UDPConn struct {
	conn      *net.UDPConn
	localAddr string // stored for DialUDP on send
	msgCh     chan<- model.Message
	lastSrc   *net.UDPAddr
	lastSrcMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	bytesSent     int64
	bytesReceived int64

	msgID uint64
	mu    sync.Mutex
}

// NewUDPConn creates a new UDP handler.
func NewUDPConn(msgCh chan<- model.Message) *UDPConn {
	return &UDPConn{
		msgCh: msgCh,
	}
}

// Bind binds to a local address and starts the read goroutine.
func (u *UDPConn) Bind(localAddr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	u.ctx = ctx
	u.cancel = cancel
	u.done = make(chan struct{})

	addr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		cancel()
		return fmt.Errorf("解析地址 %s: %w", localAddr, err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		cancel()
		return fmt.Errorf("绑定 %s: %w", localAddr, err)
	}
	u.conn = conn
	u.localAddr = localAddr

	u.msgCh <- model.Message{
		Timestamp: time.Now(),
		Direction: model.DirSystem,
		Error:     "",
		SrcAddr:   localAddr,
	}

	go u.readLoop()
	return nil
}

// SendTo sends data to the specified remote address.
// If remoteAddr is empty, sends to the last received packet's source.
// Uses a connected UDP socket (DialUDP) when a remote is specified to
// prevent the OS from broadcasting on all network interfaces.
func (u *UDPConn) SendTo(data []byte, remoteAddr string) (int, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.conn == nil {
		return 0, fmt.Errorf("UDP 未绑定")
	}

	var dstAddr *net.UDPAddr
	if remoteAddr != "" {
		var err error
		dstAddr, err = net.ResolveUDPAddr("udp", remoteAddr)
		if err != nil {
			return 0, fmt.Errorf("解析远程地址 %s: %w", remoteAddr, err)
		}
	} else {
		u.lastSrcMu.RLock()
		dstAddr = u.lastSrc
		u.lastSrcMu.RUnlock()
		if dstAddr == nil {
			return 0, fmt.Errorf("未收到过数据，请指定远程地址")
		}
	}

	n, err := u.conn.WriteToUDP(data, dstAddr)
	if err != nil {
		return n, err
	}
	u.bytesSent += int64(n)

	u.msgID++
	u.msgCh <- model.Message{
		ID:        u.msgID,
		Timestamp: time.Now(),
		Direction: model.DirSent,
		Data:      data[:n],
		DstAddr:   dstAddr.String(),
		Size:      n,
	}
	return n, nil
}

// Close closes the UDP connection.
func (u *UDPConn) Close() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.cancel != nil {
		u.cancel()
	}
	if u.conn != nil {
		u.conn.Close()
		u.conn = nil
	}
}

func (u *UDPConn) readLoop() {
	defer close(u.done)

	buf := make([]byte, 65535)
	for {
		select {
		case <-u.ctx.Done():
			return
		default:
		}

		u.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, src, err := u.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-u.ctx.Done():
				return
			default:
			}
			continue
		}

		if n > 0 {
			u.lastSrcMu.Lock()
			u.lastSrc = src
			u.lastSrcMu.Unlock()

			u.bytesReceived += int64(n)
			data := make([]byte, n)
			copy(data, buf[:n])

			u.mu.Lock()
			u.msgID++
			id := u.msgID
			u.mu.Unlock()

			u.msgCh <- model.Message{
				ID:        id,
				Timestamp: time.Now(),
				Direction: model.DirReceived,
				Data:      data,
				SrcAddr:   src.String(),
				Size:      n,
			}
		}
	}
}
