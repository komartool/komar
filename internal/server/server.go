package server

import (
	"bufio"
	"fmt"
	"komar/internal/writer"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const maxRecordBytes = 16 << 20

type Server struct {
	ln      net.Listener
	out     *writer.Writer
	wg      sync.WaitGroup
	mu      sync.Mutex
	conns   map[net.Conn]struct{}
	invalid atomic.Int64
}

func Listen(out *writer.Writer) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for agents: %w", err)
	}
	s := &Server{ln: ln, out: out, conns: map[net.Conn]struct{}{}}
	s.wg.Add(1)
	go s.accept()
	return s, nil
}

func (s *Server) Addr() *net.TCPAddr {
	addr, _ := s.ln.Addr().(*net.TCPAddr)
	return addr
}

func (s *Server) Invalid() int64 {
	return s.invalid.Load()
}

func (s *Server) Shutdown(grace time.Duration) {
	_ = s.ln.Close()
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(grace):
	}
	s.mu.Lock()
	for c := range s.conns {
		_ = c.Close()
	}
	s.mu.Unlock()
	<-done
}

func (s *Server) accept() {
	defer s.wg.Done()
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[c] = struct{}{}
		s.mu.Unlock()
		s.wg.Add(1)
		go s.read(c)
	}
}

func (s *Server) read(c net.Conn) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		delete(s.conns, c)
		s.mu.Unlock()
		_ = c.Close()
	}()
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 64<<10), maxRecordBytes)
	for sc.Scan() {
		if err := s.out.Forward(sc.Bytes()); err != nil {
			s.invalid.Add(1)
		}
	}
}
