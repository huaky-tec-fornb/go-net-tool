package model

import "time"

// Direction indicates whether a message was sent, received, or is a system event.
type Direction int

const (
	DirSent     Direction = iota // outgoing data
	DirReceived                   // incoming data
	DirSystem                     // system messages (connection events, errors)
)

// Message represents a single data or system event.
type Message struct {
	ID        uint64
	Timestamp time.Time
	Direction Direction
	Data      []byte // raw bytes
	SrcAddr   string // source IP:port
	DstAddr   string // destination IP:port
	Size      int    // byte count
	ClientID  string // TCP server client identifier
	Error     string // non-empty for error messages
}
