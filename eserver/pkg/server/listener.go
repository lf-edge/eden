package server

import (
	"bufio"
	"errors"
	"log"
	"net"
	"time"
)

type listener struct {
	accept chan net.Conn
	net.Listener
}

func newListener(l net.Listener) *listener {
	return &listener{
		make(chan net.Conn),
		l,
	}
}

// Accept connection on listener
func (l *listener) Accept() (net.Conn, error) {
	if l.accept == nil {
		return nil, errors.New("listener closed")
	}
	return <-l.accept, nil
}

// Close connection on listener
func (l *listener) Close() error {
	close(l.accept)
	l.accept = nil
	return nil
}

type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

// Peek n bytes on connection
func (b bufferedConn) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

// Read n bytes on connection
func (b bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

// MuxListener creates two net.Listener one of them accepts connections that start with "SSH",
// and another that accepts all others.
func MuxListener(l net.Listener) (ssh net.Listener, other net.Listener) {
	sshListener, otherListener := newListener(l), newListener(l)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("Error accepting conn:", err)
				continue
			}
			if err := conn.SetReadDeadline(time.Now().Add(time.Second * 10)); err != nil {
				log.Println("Error SetReadDeadline:", err)
				continue
			}
			bconn := bufferedConn{conn, bufio.NewReaderSize(conn, 3)}
			p, err := bconn.Peek(3)
			if err != nil {
				log.Println("Error peeking into conn:", err)
				continue
			}
			if err := conn.SetReadDeadline(time.Time{}); err != nil {
				log.Println("Error SetReadDeadline:", err)
				continue
			}
			prefix := string(p)
			selectedListener := otherListener
			if prefix == "SSH" {
				selectedListener = sshListener
			}
			if selectedListener.accept != nil {
				selectedListener.accept <- bconn
			}
		}
	}()
	return sshListener, otherListener
}
