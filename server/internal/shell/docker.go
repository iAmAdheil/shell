package shell

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Image is what a Session runs. Debian ships real bash and the coreutils a
// visitor expects, which matters more here than a smaller image would.
const Image = "debian:bookworm-slim"

// LabelKey marks every container this server starts, so its own containers
// can be told apart from anything else on the host.
const LabelKey = "com.shell.session"

// Resource limits per Session. A portfolio demo does not need a large box, and
// a capped container cannot take the host down with a fork bomb or a big
// allocation.
const (
	memoryBytes = 512 * 1024 * 1024
	nanoCPUs    = 500_000_000 // half a core
	maxPIDs     = 256
)

// DockerRunner starts each Session's shell in its own container.
type DockerRunner struct {
	cli *client.Client
}

func NewDockerRunner() (*DockerRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connect to docker: %w", err)
	}
	if _, err := cli.Ping(context.Background()); err != nil {
		cli.Close()
		return nil, fmt.Errorf("ping docker: %w", err)
	}
	return &DockerRunner{cli: cli}, nil
}

// Start creates a container, attaches to its terminal, and runs bash in it.
//
// Docker allocates the PTY itself: Tty makes the attached stream a raw
// terminal, so there is no separate PTY to manage on this side.
func (r *DockerRunner) Start(ctx context.Context, rows, cols uint16) (Shell, error) {
	pids := int64(maxPIDs)

	created, err := r.cli.ContainerCreate(ctx,
		&container.Config{
			Image:        Image,
			Cmd:          []string{"bash"},
			Env:          []string{"TERM=xterm-256color"},
			WorkingDir:   "/root",
			Labels:       map[string]string{LabelKey: "1"},
			Tty:          true,
			OpenStdin:    true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory:    memoryBytes,
				NanoCPUs:  nanoCPUs,
				PidsLimit: &pids,
			},
		},
		nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	// Attach before start, so no output is lost between the two calls.
	attached, err := r.cli.ContainerAttach(ctx, created.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		r.remove(created.ID)
		return nil, fmt.Errorf("attach to container: %w", err)
	}

	if err := r.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		attached.Close()
		r.remove(created.ID)
		return nil, fmt.Errorf("start container: %w", err)
	}

	sh := &dockerShell{cli: r.cli, id: created.ID, conn: attached}
	if err := sh.Resize(rows, cols); err != nil {
		sh.Close()
		return nil, fmt.Errorf("size the terminal: %w", err)
	}
	return sh, nil
}

// ContainerExists reports whether a container is still known to Docker.
func (r *DockerRunner) ContainerExists(ctx context.Context, id string) bool {
	_, err := r.cli.ContainerInspect(ctx, id)
	return err == nil
}

func (r *DockerRunner) remove(id string) {
	_ = r.cli.ContainerRemove(context.Background(), id, container.RemoveOptions{Force: true})
}

// dockerShell is one container's terminal.
type dockerShell struct {
	cli  *client.Client
	id   string
	conn types.HijackedResponse

	closeOnce sync.Once
	closeErr  error
}

func (s *dockerShell) Read(p []byte) (int, error)  { return s.conn.Reader.Read(p) }
func (s *dockerShell) Write(p []byte) (int, error) { return s.conn.Conn.Write(p) }

func (s *dockerShell) Resize(rows, cols uint16) error {
	err := s.cli.ContainerResize(context.Background(), s.id, container.ResizeOptions{
		Height: uint(rows),
		Width:  uint(cols),
	})
	if err != nil {
		return fmt.Errorf("resize container terminal: %w", err)
	}
	return nil
}

// Close drops the connection and destroys the container. A Session's container
// must not outlive the Session (ticket 20).
func (s *dockerShell) Close() error {
	s.closeOnce.Do(func() {
		s.conn.Close()

		err := s.cli.ContainerRemove(context.Background(), s.id, container.RemoveOptions{Force: true})
		if err != nil && !client.IsErrNotFound(err) {
			s.closeErr = fmt.Errorf("remove container %s: %w", s.id, err)
		}
	})
	return s.closeErr
}

// ContainerID reports which container this shell runs in.
func (s *dockerShell) ContainerID() string { return s.id }

var _ Runner = (*DockerRunner)(nil)
var _ Shell = (*dockerShell)(nil)
