package shell_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"backend/internal/shell"
)

// These tests start real containers. They are the only proof that the Docker
// runner works: everything above it is tested against a fake.

func newRunner(t *testing.T) *shell.DockerRunner {
	t.Helper()

	if os.Getenv("SKIP_DOCKER_TESTS") != "" {
		t.Skip("SKIP_DOCKER_TESTS is set")
	}

	runner, err := shell.NewDockerRunner()
	if err != nil {
		t.Skipf("no Docker to talk to: %v", err)
	}
	return runner
}

// readUntil collects output until it holds want, or gives up.
func readUntil(t *testing.T, sh shell.Shell, want string) string {
	t.Helper()

	var seen strings.Builder
	done := make(chan struct{})

	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := sh.Read(buf)
			if n > 0 {
				seen.Write(buf[:n])
				if strings.Contains(seen.String(), want) {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("timed out waiting for %q, saw: %q", want, seen.String())
	}
	return seen.String()
}

func TestARealShellRunsARealCommand(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()

	sh, err := runner.Start(ctx, 24, 80)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { sh.Close() })

	if _, err := sh.Write([]byte("echo shell-is-alive\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := readUntil(t, sh, "shell-is-alive"); !strings.Contains(got, "shell-is-alive") {
		t.Errorf("never saw the command output, saw: %q", got)
	}
}

func TestTheShellIsIsolatedInItsOwnContainer(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()

	sh, err := runner.Start(ctx, 24, 80)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { sh.Close() })

	// The host is macOS. Seeing Linux proves the command ran inside a
	// container, not on the machine running the test.
	if _, err := sh.Write([]byte("uname -s\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := readUntil(t, sh, "Linux"); !strings.Contains(got, "Linux") {
		t.Errorf("expected a Linux container, saw: %q", got)
	}
}

func TestClosingAShellDestroysItsContainer(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()

	sh, err := runner.Start(ctx, 24, 80)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	id := sh.(interface{ ContainerID() string }).ContainerID()

	if err := sh.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if runner.ContainerExists(ctx, id) {
		t.Errorf("container %s is still there after Close", id[:12])
	}
}

func TestASessionRunsMyShellByDefault(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()

	sh, err := runner.Start(ctx, 24, 80)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { sh.Close() })

	// PID 1 is the container's default command.
	if _, err := sh.Write([]byte("readlink /proc/1/exe\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	want := "/usr/local/bin/my-shell"
	if got := readUntil(t, sh, want); !strings.Contains(got, want) {
		t.Errorf("my-shell is not the default command, saw: %q", got)
	}
}

func TestASessionHasAptPackageLists(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()

	sh, err := runner.Start(ctx, 24, 80)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { sh.Close() })

	// A list file is there only after an apt-get update. Without one, a User
	// cannot install a package.
	if _, err := sh.Write([]byte("ls /var/lib/apt/lists\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := readUntil(t, sh, "_Packages"); !strings.Contains(got, "_Packages") {
		t.Errorf("no apt package lists in the image, saw: %q", got)
	}
}

func TestTheShellStartsAtTheRequestedSize(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()

	sh, err := runner.Start(ctx, 30, 100)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { sh.Close() })

	if _, err := sh.Write([]byte("stty -F /dev/tty size\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := readUntil(t, sh, "30 100"); !strings.Contains(got, "30 100") {
		t.Errorf("terminal is not 30x100, saw: %q", got)
	}
}

func TestResizingChangesTheTerminalSize(t *testing.T) {
	runner := newRunner(t)
	ctx := context.Background()

	sh, err := runner.Start(ctx, 24, 80)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { sh.Close() })
	readUntil(t, sh, "$ ")

	if err := sh.Resize(40, 120); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	if _, err := sh.Write([]byte("stty -F /dev/tty size\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := readUntil(t, sh, "40 120"); !strings.Contains(got, "40 120") {
		t.Errorf("terminal did not resize, saw: %q", got)
	}
}
