package model

// Protocol is the network protocol mode.
type Protocol string

const (
	ProtoTCPClient Protocol = "tcp-client"
	ProtoTCPServer Protocol = "tcp-server"
	ProtoUDP       Protocol = "udp"
)

// ConnectionState represents the current connection status.
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateConnected    ConnectionState = "connected"
	StateError        ConnectionState = "error"
)

// ConnectionConfig holds all user-configured connection parameters.
type ConnectionConfig struct {
	Protocol   Protocol `json:"protocol"`
	LocalIP    string   `json:"localIp"`
	LocalPort  int      `json:"localPort"`
	RemoteIP   string   `json:"remoteIp"`
	RemotePort int      `json:"remotePort"`
}

// ClientInfo describes a connected TCP client (server mode).
type ClientInfo struct {
	ID            string `json:"id"`
	RemoteAddr    string `json:"remoteAddr"`
	ConnectedAt   string `json:"connectedAt"`
	BytesSent     int64  `json:"bytesSent"`
	BytesReceived int64  `json:"bytesReceived"`
}

// ByteCounters tracks sent/received statistics.
type ByteCounters struct {
	BytesSent     int64 `json:"bytesSent"`
	BytesReceived int64 `json:"bytesReceived"`
	PacketsSent   int64 `json:"packetsSent"`
	PacketsReceived int64 `json:"packetsReceived"`
}
