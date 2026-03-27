package container

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) List() ([]ContainerInfo, error) {
	out, err := exec.Command("docker", "ps", "-a",
		"--format", `{"id":"{{.ID}}","name":"{{.Names}}","image":"{{.Image}}","state":"{{.State}}","status":"{{.Status}}","ports":"{{.Ports}}","created":"{{.CreatedAt}}"}`).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %s", string(out))
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var c ContainerInfo
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func (s *Service) Start(id string) error {
	out, err := exec.Command("docker", "start", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("start: %s", string(out))
	}
	return nil
}

func (s *Service) Stop(id string) error {
	out, err := exec.Command("docker", "stop", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop: %s", string(out))
	}
	return nil
}

func (s *Service) Restart(id string) error {
	out, err := exec.Command("docker", "restart", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart: %s", string(out))
	}
	return nil
}

func (s *Service) Remove(id string) error {
	exec.Command("docker", "stop", id).Run()
	out, err := exec.Command("docker", "rm", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove: %s", string(out))
	}
	return nil
}

func (s *Service) Logs(id string, tail int) (string, error) {
	if tail <= 0 {
		tail = 100
	}
	out, err := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", tail), id).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("logs: %s", string(out))
	}
	return string(out), nil
}
