package ygggo_mysql

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner abstracts shell command execution for testability
// It returns stdout, stderr, exitCode and an error if command could not be started.
// If err == nil, exitCode should still be checked for non-zero failures.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, string, int, error)
}

type defaultCommandRunner struct{}

func (d defaultCommandRunner) Run(ctx context.Context, name string, args ...string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	// exec doesn't expose exit code portably without using ExitError
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	}
	return string(out), "", code, nil
}

// dockerRunner is package-level overridable runner (for tests)
var dockerRunner CommandRunner = defaultCommandRunner{}

// env helpers
func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// Docker/MySQL related environment keys
const (
	EnvDockerContainerName = "YGGGO_MYSQL_DOCKER_CONTAINER"
	EnvMySQLVersion        = "YGGGO_MYSQL_VERSION"
	EnvMySQLDatabase       = "YGGGO_MYSQL_DATABASE"
	EnvMySQLUser           = "YGGGO_MYSQL_USERNAME"
	EnvMySQLPassword       = "YGGGO_MYSQL_PASSWORD"
	EnvMySQLRootPassword   = "YGGGO_MYSQL_ROOT_PASSWORD"
	EnvMySQLPort           = "YGGGO_MYSQL_PORT" // host port to bind 3306 to
)

// IsDockerInstalled checks if Docker CLI is available on the system
func IsDockerInstalled(ctx context.Context) bool {
	out, _, code, _ := dockerRunner.Run(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	return code == 0 && strings.TrimSpace(out) != ""
}

// IsMySQL checks whether the configured MySQL container exists and is running
func IsMySQL(ctx context.Context) bool {
	name := getenv(EnvDockerContainerName, "ygggo-mysql")
	// inspect running state
	out, _, code, _ := dockerRunner.Run(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name)
	if code == 0 {
		return strings.HasPrefix(strings.TrimSpace(strings.ToLower(out)), "true")
	}
	return false
}

// resolveMappedPort reads the host port mapped to container 3306 and updates env
func resolveMappedPort(ctx context.Context, name string) string {
	out, _, code, _ := dockerRunner.Run(ctx, "docker", "inspect", "-f", "{{ (index (index .NetworkSettings.Ports \"3306/tcp\") 0).HostPort }}", name)
	if code != 0 {
		return ""
	}
	return strings.TrimSpace(out)
}

// NewMySQL ensures a MySQL container exists (creating if needed) using environment configuration
// Required/Used env vars:
// - YGGGO_MYSQL_DOCKER_CONTAINER (default: ygggo-mysql)
// - YGGGO_MYSQL_VERSION (default: 8.0)
// - YGGGO_MYSQL_DATABASE, YGGGO_MYSQL_USERNAME, YGGGO_MYSQL_PASSWORD
// - YGGGO_MYSQL_ROOT_PASSWORD (default: rootpass)
// - YGGGO_MYSQL_PORT (default: 3306)
func NewMySQL(ctx context.Context) error {
	if !IsDockerInstalled(ctx) {
		return fmt.Errorf("docker not installed or not available in PATH")
	}

	name := getenv(EnvDockerContainerName, "ygggo-mysql")
	version := getenv(EnvMySQLVersion, "8.0")
	database := getenv(EnvMySQLDatabase, "testdb")
	user := getenv(EnvMySQLUser, "testuser")
	password := getenv(EnvMySQLPassword, "testpass")
	rootPassword := getenv(EnvMySQLRootPassword, "")
	port := getenv(EnvMySQLPort, "3306")
	allowEmpty := boolish(os.Getenv("MYSQL_ALLOW_EMPTY_PASSWORD")) || boolish(os.Getenv("YGGGO_MYSQL_ALLOW_EMPTY_PASSWORD"))
	if strings.ToLower(strings.TrimSpace(user)) == "root" && !allowEmpty {
		if strings.TrimSpace(rootPassword) == "" {
			// Fallback: use YGGGO_MYSQL_PASSWORD as root password when explicit root password not provided
			rootPassword = password
		}
	}

	// If already running, nothing to do
	if IsMySQL(ctx) {
		return nil
	}

	// If container exists but not running -> start it
	if containerExists(ctx, name) {
		_, _, code, _ := dockerRunner.Run(ctx, "docker", "start", name)
		if code != 0 {
			return fmt.Errorf("failed to start existing container %s", name)
		}
		if p := resolveMappedPort(ctx, name); p != "" {
			_ = os.Setenv(EnvMySQLPort, p)
		}
		// Align password env for root user case
		if strings.ToLower(strings.TrimSpace(user)) == "root" {
			if allowEmpty {
				_ = os.Setenv("YGGGO_MYSQL_PASSWORD", "")
			} else {
				_ = os.Setenv("YGGGO_MYSQL_PASSWORD", rootPassword)
			}
		}
		return nil
	}

	// Pre-pull image to improve reliability and clearer error logs
	_, _, _, _ = dockerRunner.Run(ctx, "docker", "pull", "mysql:"+version)

	// Otherwise, run new container
	publish := fmt.Sprintf("%s:3306", port)
	if strings.TrimSpace(port) == "" || strings.TrimSpace(port) == "0" {
		publish = "3306" // let docker choose an ephemeral host port
	}
	args := []string{
		"run", "-d",
		"--name", name,
		"-v", fmt.Sprintf("%s_data:/var/lib/mysql", name),
		"-e", "MYSQL_DATABASE=" + database,
	}
	// Do not set MYSQL_USER/MYSQL_PASSWORD when user is root (official image forbids this)
	if strings.ToLower(strings.TrimSpace(user)) != "root" {
		args = append(args,
			"-e", "MYSQL_USER="+user,
			"-e", "MYSQL_PASSWORD="+password,
		)
	} else {
		// Root auth strategy
		if allowEmpty {
			args = append(args, "-e", "MYSQL_ALLOW_EMPTY_PASSWORD=yes")
			_ = os.Setenv("YGGGO_MYSQL_PASSWORD", "")
		} else {
			args = append(args, "-e", "MYSQL_ROOT_PASSWORD="+rootPassword)
			_ = os.Setenv("YGGGO_MYSQL_PASSWORD", rootPassword)
		}
		// Allow root to connect from any host (needed for host->container connections)
		args = append(args, "-e", "MYSQL_ROOT_HOST=%")
	}
	args = append(args,
		"-p", publish,
		"mysql:"+version,
	)
	out, errOut, code, _ := dockerRunner.Run(ctx, "docker", args...)
	if code != 0 {
		msg := strings.TrimSpace(out)
		if errOut != "" {
			msg = strings.TrimSpace(errOut)
		}
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("failed to run mysql docker container: %s", msg)
	}
	// If user provided port=0 or default failed, ensure env port reflects actual mapping
	if p := resolveMappedPort(ctx, name); p != "" {
		_ = os.Setenv(EnvMySQLPort, p)
	}
	// Ensure host is set for connection
	if strings.TrimSpace(os.Getenv("YGGGO_MYSQL_HOST")) == "" {
		_ = os.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	}

	// Wait for MySQL to be ready (important for fresh containers)
	return waitForMySQLReady(ctx, name, 60*time.Second)
}

// waitForMySQLReady waits for MySQL container to be ready to accept connections
func waitForMySQLReady(ctx context.Context, containerName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for MySQL to be ready: %w", ctx.Err())
		case <-ticker.C:
			// Check if container is still running
			out, _, code, _ := dockerRunner.Run(ctx, "docker", "inspect", "-f", "{{.State.Running}}", containerName)
			if code != 0 || !strings.HasPrefix(strings.TrimSpace(strings.ToLower(out)), "true") {
				return fmt.Errorf("MySQL container stopped unexpectedly")
			}

			// Try to connect using mysqladmin ping
			_, _, pingCode, _ := dockerRunner.Run(ctx, "docker", "exec", containerName, "mysqladmin", "ping", "-h", "localhost", "--silent")
			if pingCode == 0 {
				return nil // MySQL is ready
			}
		}
	}
}

// DeleteMySQL stops and removes the configured MySQL container if it exists
func DeleteMySQL(ctx context.Context) error {
	name := getenv(EnvDockerContainerName, "ygggo-mysql")
	if !containerExists(ctx, name) {
		return nil
	}
	_, _, code, _ := dockerRunner.Run(ctx, "docker", "rm", "-f", "-v", name)
	if code != 0 {
		return fmt.Errorf("failed to remove container %s", name)
	}
	// attempt to remove named volume (ignore failures)
	_, _, _, _ = dockerRunner.Run(ctx, "docker", "volume", "rm", fmt.Sprintf("%s_data", name))
	// small grace period for cleanup on some systems
	time.Sleep(200 * time.Millisecond)
	return nil
}

func containerExists(ctx context.Context, name string) bool {
	// docker inspect returns non-zero exit when container doesn't exist
	_, _, code, _ := dockerRunner.Run(ctx, "docker", "inspect", name)
	return code == 0
}

func atoiSafe(s string) int {
	var n int
	_, _ = fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n
}

func boolish(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "1" || v == "true" || v == "yes" || v == "y" || v == "on"
}
