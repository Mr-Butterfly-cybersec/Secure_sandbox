package traps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

type FileTrap struct {
	Path string
}

func NewFileTrap() (*FileTrap, error) {
	tmpDir := os.TempDir()
	fifoPath := filepath.Join(tmpDir, fmt.Sprintf("trap_%d", syscall.Getpid()))
	
	// Remove if exists
	os.Remove(fifoPath)

	err := unix.Mkfifo(fifoPath, 0666)
	if err != nil {
		return nil, err
	}

	return &FileTrap{Path: fifoPath}, nil
}

func (f *FileTrap) Watch(ctx context.Context, trigger chan bool) {
	// Opening a FIFO O_WRONLY blocks until a reader opens it
	// We use a goroutine so we can cancel it if the container finishes normally
	go func() {
		fd, err := unix.Open(f.Path, unix.O_WRONLY|unix.O_NONBLOCK, 0)
		if err == nil {
			unix.Close(fd)
		}

		// Since we used O_NONBLOCK to check if it's already open, we actually want 
		// the blocking behavior to detect the "event".
		// Correct approach for detection:
		file, err := os.OpenFile(f.Path, os.O_WRONLY, 0)
		if err == nil {
			trigger <- true
			file.Close()
		}
	}()
}

func (f *FileTrap) Cleanup() {
	os.Remove(f.Path)
}
