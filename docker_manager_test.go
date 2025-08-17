package ygggo_mysql

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fakeRunner struct{ script []string }

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, string, int, error) {
	cmd := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.script = append(f.script, cmd)
	switch {
	case strings.HasPrefix(cmd, "docker version"):
		return "24.0.0", "", 0, nil
	case strings.HasPrefix(cmd, "docker inspect"):
		// not running yet
		return "false", "", 0, nil
	case strings.HasPrefix(cmd, "docker inspect "):
		// container does not exist -> non-zero simulated by returning empty but tests don't check code here
		return "", "", 1, nil
	case strings.HasPrefix(cmd, "docker pull mysql:"):
		return "pulled", "", 0, nil
	case strings.HasPrefix(cmd, "docker run"):
		return "created", "", 0, nil
	default:
		return "", "", 0, nil
	}
}

func TestDockerManager_NewMySQL_CreatesContainer(t *testing.T) {
	ctx := context.Background()
	old := dockerRunner
	fr := &fakeRunner{}
	dockerRunner = fr
	t.Cleanup(func() { dockerRunner = old })

	t.Setenv(EnvDockerContainerName, "ygggo-mysql-test")
	t.Setenv(EnvMySQLVersion, "8.0")
	t.Setenv(EnvMySQLDatabase, "testdb")
	t.Setenv(EnvMySQLUser, "testuser")
	t.Setenv(EnvMySQLPassword, "testpass")
	t.Setenv(EnvMySQLRootPassword, "rootpass")
	t.Setenv(EnvMySQLPort, "3307")

	if err := NewMySQL(ctx); err != nil {
		t.Fatalf("NewMySQL err: %v", err)
	}

	foundRun := false
	for _, s := range fr.script {
		if strings.HasPrefix(s, "docker run ") && strings.Contains(s, "-p 3307:3306") {
			foundRun = true
		}
	}
	if !foundRun {
		t.Fatalf("expected docker run with port binding, got script: %v", fr.script)
	}
}

type fakeRunnerExisting struct{ script []string }

func (f *fakeRunnerExisting) Run(ctx context.Context, name string, args ...string) (string, string, int, error) {
	cmd := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.script = append(f.script, cmd)
	switch {
	case strings.HasPrefix(cmd, "docker version"):
		return "24.0.0", "", 0, nil
	case strings.HasPrefix(cmd, "docker inspect"):
		return "false", "", 0, nil
	case strings.HasPrefix(cmd, "docker inspect "):
		// exists
		return "{}", "", 0, nil
	case strings.HasPrefix(cmd, "docker start"):
		return "started", "", 0, nil
	default:
		return "", "", 0, nil
	}
}

func TestDockerManager_StartExisting(t *testing.T) {
	ctx := context.Background()
	old := dockerRunner
	fr := &fakeRunnerExisting{}
	dockerRunner = fr
	t.Cleanup(func() { dockerRunner = old })

	if err := NewMySQL(ctx); err != nil {
		t.Fatalf("NewMySQL err: %v", err)
	}

	foundStart := false
	for _, s := range fr.script {
		if strings.HasPrefix(s, "docker start ") {
			foundStart = true
		}
	}
	if !foundStart {
		t.Fatalf("expected docker start, got: %v", fr.script)
	}
}

type fakeRunnerRunning struct{ script []string }

func (f *fakeRunnerRunning) Run(ctx context.Context, name string, args ...string) (string, string, int, error) {
	cmd := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.script = append(f.script, cmd)
	switch {
	case strings.HasPrefix(cmd, "docker version"):
		return "24.0.0", "", 0, nil
	case strings.HasPrefix(cmd, "docker inspect"):
		return "true", "", 0, nil
	default:
		return "", "", 0, nil
	}
}

func TestDockerManager_IsMySQL_Running_Noop(t *testing.T) {
	ctx := context.Background()
	old := dockerRunner
	fr := &fakeRunnerRunning{}
	dockerRunner = fr
	t.Cleanup(func() { dockerRunner = old })

	if err := NewMySQL(ctx); err != nil {
		t.Fatalf("NewMySQL err: %v", err)
	}
}

type fakeRunnerDelete struct{ script []string }

func (f *fakeRunnerDelete) Run(ctx context.Context, name string, args ...string) (string, string, int, error) {
	cmd := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.script = append(f.script, cmd)
	switch {
	case strings.HasPrefix(cmd, "docker inspect "):
		return "{}", "", 0, nil
	case strings.HasPrefix(cmd, "docker rm -f"):
		return "removed", "", 0, nil
	default:
		return "", "", 0, nil
	}
}

func TestDockerManager_Delete(t *testing.T) {
	ctx := context.Background()
	old := dockerRunner
	fr := &fakeRunnerDelete{}
	dockerRunner = fr
	t.Cleanup(func() { dockerRunner = old })

	if err := DeleteMySQL(ctx); err != nil {
		t.Fatalf("DeleteMySQL err: %v", err)
	}
}

func TestIsDockerInstalled_FalseWhenNoDocker(t *testing.T) {
	ctx := context.Background()
	old := dockerRunner
	dockerRunner = CommandRunnerFunc(func(ctx context.Context, name string, args ...string) (string, string, int, error) {
		return "", "", 1, fmt.Errorf("not found")
	})
	t.Cleanup(func() { dockerRunner = old })
	if IsDockerInstalled(ctx) {
		t.Fatalf("expected false when docker not installed")
	}
}

type CommandRunnerFunc func(ctx context.Context, name string, args ...string) (string, string, int, error)

func (f CommandRunnerFunc) Run(ctx context.Context, name string, args ...string) (string, string, int, error) {
	return f(ctx, name, args...)
}
