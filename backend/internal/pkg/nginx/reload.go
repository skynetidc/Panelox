package nginx

import (
	"fmt"
	"os/exec"
)

// TestConfig runs nginx -t to validate the configuration.
func TestConfig() error {
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx config test failed: %s", string(out))
	}
	return nil
}

// Reload sends a reload signal to nginx.
func Reload() error {
	out, err := exec.Command("nginx", "-s", "reload").CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx reload failed: %s", string(out))
	}
	return nil
}

// TestAndReload validates config then reloads nginx.
func TestAndReload() error {
	if err := TestConfig(); err != nil {
		return err
	}
	return Reload()
}

// DockerExec runs an nginx command inside a Docker container.
func DockerExec(containerName, operation string) error {
	args := []string{"exec", containerName, "nginx"}
	switch operation {
	case "test":
		args = append(args, "-t")
	case "reload":
		args = append(args, "-s", "reload")
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx %s failed in container %s: %s", operation, containerName, string(out))
	}
	return nil
}
