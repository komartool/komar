package writer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
)

var ErrNotObject = errors.New("record is not a JSON object")

type Writer struct {
	mu   sync.Mutex
	file *os.File
	buf  *bufio.Writer
	seq  int64
}

func Create(path string) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create output: %w", err)
	}
	return &Writer{file: f, buf: bufio.NewWriter(f)}, nil
}

func (w *Writer) Write(record any) error {
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode record: %w", err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.line(line)
}

func (w *Writer) Forward(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '{' || !json.Valid(raw) {
		return ErrNotObject
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.seq++
	line := make([]byte, 0, len(raw)+24)
	line = append(line, raw[:len(raw)-1]...)
	if len(bytes.TrimSpace(raw[1:len(raw)-1])) > 0 {
		line = append(line, ',')
	}
	line = append(line, `"seq":`...)
	line = strconv.AppendInt(line, w.seq, 10)
	line = append(line, '}')
	return w.line(line)
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	flushErr := w.buf.Flush()
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	if flushErr != nil {
		return fmt.Errorf("flush output: %w", flushErr)
	}
	return nil
}

func (w *Writer) line(line []byte) error {
	if _, err := w.buf.Write(line); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := w.buf.WriteByte('\n'); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	return nil
}
