package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mxprshn/komar/internal/writer"
)

const (
	maxRecordBytes = 16 << 20
	kindHit        = "hit"
	kindStats      = "stats"
	kindDiag       = "diag"
	msgPlaced      = "placed"
)

type Totals struct {
	Hits       map[string]int
	WhenErrors map[string]int64
	Unplaced   map[string]string
	Dropped    int64
	Invalid    int64
}

type Server struct {
	ln        net.Listener
	out       *writer.Writer
	wg        sync.WaitGroup
	mu        sync.Mutex
	conns     map[net.Conn]struct{}
	totals    Totals
	hitLimit  int
	hitCount  int
	placed    map[string]bool
	unplaced  map[string]string
	limitOnce sync.Once
	limitHit  chan struct{}
}

func Listen(out *writer.Writer, hitLimit int) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for agents: %w", err)
	}
	s := &Server{
		ln: ln, out: out, conns: map[net.Conn]struct{}{},
		totals:   Totals{Hits: map[string]int{}, WhenErrors: map[string]int64{}},
		hitLimit: hitLimit, limitHit: make(chan struct{}),
		placed: map[string]bool{}, unplaced: map[string]string{},
	}
	s.wg.Add(1)
	go s.accept()
	return s, nil
}

func (s *Server) Addr() *net.TCPAddr {
	addr, _ := s.ln.Addr().(*net.TCPAddr)
	return addr
}

func (s *Server) LimitReached() <-chan struct{} {
	return s.limitHit
}

func (s *Server) Totals() Totals {
	s.mu.Lock()
	defer s.mu.Unlock()
	hits := make(map[string]int, len(s.totals.Hits))
	for k, v := range s.totals.Hits {
		hits[k] = v
	}
	whenErrors := make(map[string]int64, len(s.totals.WhenErrors))
	for k, v := range s.totals.WhenErrors {
		whenErrors[k] = v
	}
	unplaced := map[string]string{}
	for k, v := range s.unplaced {
		if !s.placed[k] {
			unplaced[k] = v
		}
	}
	return Totals{
		Hits: hits, WhenErrors: whenErrors, Unplaced: unplaced,
		Dropped: s.totals.Dropped, Invalid: s.totals.Invalid,
	}
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
	Kind       string            `json:"kind"`
	Probe      string            `json:"probe"`
	Msg        string            `json:"msg"`
	Dropped    int64             `json:"dropped"`
	WhenErrors map[string]int64  `json:"when_errors"`
	Unplaced   map[string]string `json:"unplaced"`
}

func (s *Server) handle(line []byte) {
	var h header
	if err := json.Unmarshal(line, &h); err != nil {
		s.count(func(t *Totals) { t.Invalid++ })
		return
	}
	if h.Kind == kindStats {
		s.count(func(t *Totals) { s.mergeStats(t, h) })
		return
	}
	if h.Kind == kindDiag && h.Msg == msgPlaced {
		s.count(func(*Totals) { s.placed[h.Probe] = true })
	}
	if h.Kind == kindHit && !s.admitHit(h.Probe) {
		return
	}
	if err := s.out.Forward(line); err != nil {
		s.count(func(t *Totals) { t.Invalid++ })
	}
}

func (s *Server) mergeStats(t *Totals, h header) {
	t.Dropped += h.Dropped
	for k, v := range h.WhenErrors {
		t.WhenErrors[k] += v
	}
	for k, v := range h.Unplaced {
		if _, seen := s.unplaced[k]; !seen {
			s.unplaced[k] = v
		}
	}
}

func (s *Server) admitHit(probe string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hitLimit > 0 && s.hitCount >= s.hitLimit {
		return false
	}
	s.hitCount++
	s.totals.Hits[probe]++
	if s.hitLimit > 0 && s.hitCount == s.hitLimit {
		s.limitOnce.Do(func() { close(s.limitHit) })
	}
	return true
}

func (s *Server) count(update func(*Totals)) {
	s.mu.Lock()
	update(&s.totals)
	s.mu.Unlock()
}
