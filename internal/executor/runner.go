package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sandbox/internal/traps"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Result struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	Duration   time.Duration
	TimedOut   bool
	Terminated bool   // True if killed by a trap
	TrapType   string // "file" or "api"
}

type Runner struct {
	cli *client.Client
}

func NewRunner() (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Runner{cli: cli}, nil
}

func (r *Runner) Run(ctx context.Context, lang string, code string, limits Limits, apiTrigger chan bool) (*Result, error) {
	startTime := time.Now()

	// 1. Setup Traps
	fileTrap, err := traps.NewFileTrap()
	if err != nil {
		return nil, fmt.Errorf("failed to create file trap: %v", err)
	}
	defer fileTrap.Cleanup()

	fileTrigger := make(chan bool, 1)
	fileTrap.Watch(ctx, fileTrigger)

	// 2. Ensure image exists
	img := "python:3.11-alpine"
	_, _, err = r.cli.ImageInspectWithRaw(ctx, img)
	if err != nil {
		reader, err := r.cli.ImagePull(ctx, img, image.PullOptions{})
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		io.Copy(io.Discard, reader)
	}

	var cmd []string
	if lang == "python" {
		cmd = []string{"python3", "-c", code}
	} else if lang == "bash" {
		cmd = []string{"/bin/sh", "-c", code}
	} else {
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	// 3. Container Config
	config := &container.Config{
		Image: img,
		Cmd:   cmd,
		User:  "1000:1000",
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:    limits.MemoryUsageBytes,
			NanoCPUs:  limits.NanoCPUs,
			PidsLimit: &limits.PIDsLimit,
		},
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		ReadonlyRootfs: true,
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   fileTrap.Path,
				Target:   "/var/run/secrets/database.txt",
				ReadOnly: true,
			},
		},
	}

	resp, err := r.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return nil, err
	}
	defer r.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	// 4. Start Execution
	runCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()

	if err := r.cli.ContainerStart(runCtx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

	statusCh, errCh := r.cli.ContainerWait(runCtx, resp.ID, container.WaitConditionNotRunning)
	
	var exitCode int64
	var timedOut bool
	var terminated bool
	var trapType string

	select {
	case <-fileTrigger:
		terminated = true
		trapType = "file"
		r.cli.ContainerKill(ctx, resp.ID, "SIGKILL")
	case <-apiTrigger:
		terminated = true
		trapType = "api"
		r.cli.ContainerKill(ctx, resp.ID, "SIGKILL")
	case err := <-errCh:
		if err != nil && runCtx.Err() == context.DeadlineExceeded {
			timedOut = true
		}
	case status := <-statusCh:
		exitCode = status.StatusCode
	case <-runCtx.Done():
		if runCtx.Err() == context.DeadlineExceeded {
			timedOut = true
			r.cli.ContainerKill(ctx, resp.ID, "SIGKILL")
		}
	}

	// 5. Capture Logs
	out, err := r.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	var stdout, stderr bytes.Buffer
	if err == nil {
		defer out.Close()
		stdcopy.StdCopy(&stdout, &stderr, out)
	}

	return &Result{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   int(exitCode),
		Duration:   time.Since(startTime),
		TimedOut:   timedOut,
		Terminated: terminated,
		TrapType:   trapType,
	}, nil
}

