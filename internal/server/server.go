package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"komar/internal/writer"
	"net"
	"sync"
	"time"
)

const (
	maxRecordBytes = 16 << 20
	kindHit        = "hit"
	kindStats      = "stats"
)

type Totals struct {
	Hits    map[string]int
	Dropped int64
	Invalid int64
}

type Server struct {
	ln     net.Listener
	out    *writer.Writer
	wg     sync.WaitGroup
	mu     sync.Mutex
	conns  map[net.Conn]struct{}
	totals Totals
}

func Listen(out *writer.Writer) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for agents: %w", err)
	}
	s := &Server{ln: ln, out: out, conns: map[net.Conn]struct{}{}, totals: Totals{Hits: map[string]int{}}}
	s.wg.Add(1)
	go s.accept()
	return s, nil
}

func (s *Server) Addr() *net.TCPAddr {
	addr, _ := s.ln.Addr().(*net.TCPAddr)
	return addr
}

func (s *Server) Totals() Totals {
	s.mu.Lock()
	defer s.mu.Unlock()
	hits := make(map[string]int, len(s.totals.Hits))
	for k, v := range s.totals.Hits {
		hits[k] = v
	}
	return Totals{Hits: hits, Dropped: s.totals.Dropped, Invalid: s.totals.Invalid}
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
		s.handle(sc.Bytes())
	}
}

type header struct {
	Kind    string `json:"kind"`
	Probe   string `json:"probe"`
	Dropped int64  `json:"dropped"`
}

func (s *Server) handle(line []byte) {
	var h header
	if err := json.Unmarshal(line, &h); err != nil {
		s.count(func(t *Totals) { t.Invalid++ })
		return
	}
	if h.Kind == kindStats {
		s.count(func(t *Totals) { t.Dropped += h.Dropped })
		return
	}
	if err := s.out.Forward(line); err != nil {
		s.count(func(t *Totals) { t.Invalid++ })
		return
	}
	if h.Kind == kindHit {
		s.count(func(t *Totals) { t.Hits[h.Probe]++ })
	}
}

func (s *Server) count(update func(*Totals)) {
	s.mu.Lock()
	update(&s.totals)
	s.mu.Unlock()
}
