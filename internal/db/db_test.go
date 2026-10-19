package db

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
)

type captureLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *captureLogger) Print(v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprint(v...))
}

func TestConnectionRoutesDriverDiagnosticsToConfiguredLogger(t *testing.T) {
	// Dropping the handshake triggers a real MySQL driver diagnostic, without
	// requiring a running database server or changing the global driver logger.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	host, portText, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portText)
	logger := &captureLogger{}
	_, err = NewConnection(Config{Host: host, Port: port, Charset: "utf8mb4", SSLMode: "off", Logger: logger})
	if err == nil {
		t.Fatal("expected failed handshake")
	}
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if len(logger.messages) == 0 {
		t.Fatal("driver diagnostics were not routed to the configured logger")
	}
}
