package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type inspection struct {
	Config struct {
		User string
		Env  []string
	}
	HostConfig struct {
		NetworkMode    string
		ReadonlyRootfs bool
		CapDrop        []string
		SecurityOpt    []string
	}
	Mounts []struct {
		Type, Source, Destination string
		RW                        bool
	}
}

func checkInspection(info inspection, bundle string) error {
	h := info.HostConfig
	if info.Config.User != "spike" || h.NetworkMode != "none" || !h.ReadonlyRootfs || len(h.CapDrop) != 1 || h.CapDrop[0] != "ALL" {
		return fmt.Errorf("unsafe container user, network, filesystem, or capabilities")
	}
	found := false
	for _, s := range h.SecurityOpt {
		if s == "no-new-privileges:true" {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("missing no-new-privileges")
	}
	if len(info.Mounts) != 1 {
		return fmt.Errorf("unexpected mounts")
	}
	m := info.Mounts[0]
	source, err := filepath.EvalSymlinks(m.Source)
	if err != nil {
		return err
	}
	expected, err := filepath.EvalSymlinks(filepath.Join(bundle, "runtime"))
	if err != nil {
		return err
	}
	if m.Type != "bind" || m.Destination != "/spike" || m.RW || source != expected {
		return fmt.Errorf("unsafe runtime mount")
	}
	for _, s := range info.Config.Env {
		if forbiddenEnv(strings.SplitN(s, "=", 2)[0]) {
			return fmt.Errorf("credential environment present")
		}
	}
	return nil
}
func forbiddenEnv(key string) bool {
	switch key {
	case "GH_TOKEN", "GITHUB_TOKEN", "SSH_AUTH_SOCK", "AWS_SECRET_ACCESS_KEY":
		return true
	}
	return false
}
func isolate(bundle string) (err error) {
	if _, err = os.Stat(filepath.Join(bundle, "runtime/bin/advisory")); err != nil {
		return err
	}
	repo, err := root()
	if err != nil {
		return err
	}
	home, err := os.MkdirTemp("", "renodash-devcontainer-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)
	canary := filepath.Join(home, "host-canary")
	if err = os.WriteFile(canary, []byte("synthetic host secret; never mount"), 0600); err != nil {
		return err
	}
	env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "RENODASH_SPIKE_BUNDLE=" + bundle}
	out, startErr := output("", env, "devcontainer", "up", "--workspace-folder", bundle, "--config", filepath.Join(repo, "spikes/local-skill/.devcontainer/devcontainer.json"), "--mount-workspace-git-root", "false", "--skip-post-create", "--container-session-data-folder", "/tmp/devcontainer-session")
	var result struct {
		ContainerID string `json:"containerId"`
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	parseErr := json.Unmarshal([]byte(lines[len(lines)-1]), &result)
	if result.ContainerID != "" {
		defer func() {
			_, cleanupErr := output("", nil, "docker", "rm", "-f", result.ContainerID)
			err = errors.Join(err, cleanupErr)
		}()
	}
	if startErr != nil {
		return startErr
	}
	if parseErr != nil {
		return parseErr
	}
	if result.ContainerID == "" {
		return fmt.Errorf("missing container ID")
	}
	raw, err := output("", nil, "docker", "inspect", result.ContainerID)
	if err != nil {
		return err
	}
	var infos []inspection
	if err = json.Unmarshal(raw, &infos); err != nil {
		return err
	}
	if len(infos) != 1 {
		return fmt.Errorf("unexpected inspect result")
	}
	if err = checkInspection(infos[0], bundle); err != nil {
		return err
	}
	out, err = output("", nil, "docker", "exec", result.ContainerID, "/spike/bin/spike", "probe")
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	if _, err = output("", nil, "docker", "exec", result.ContainerID, "test", "!", "-e", canary); err != nil {
		return err
	}
	fmt.Println("Effective Dev Container configuration and non-shared host canary: PASS")
	return nil
}
func probe() error {
	if os.Getuid() == 0 {
		return fmt.Errorf("root UID")
	}
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return err
	}
	status := string(data)
	if !strings.Contains(status, "CapEff:\t0000000000000000") || !strings.Contains(status, "NoNewPrivs:\t1") {
		return fmt.Errorf("capability or elevation restriction missing")
	}
	for _, path := range []string{"/spike/provenance.json", "/etc/renodash-probe"} {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			_, writeErr := f.Write([]byte("forbidden"))
			closeErr := f.Close()
			err = errors.Join(writeErr, closeErr)
		}
		if !errors.Is(err, syscall.EROFS) && !errors.Is(err, syscall.EACCES) {
			return fmt.Errorf("write probe %s: expected EROFS or EACCES, got %v", path, err)
		}
	}
	for _, path := range []string{"/var/run/docker.sock", "/host-canary", "/root/.ssh", "/root/.config/gh"} {
		_, err := os.Stat(path)
		if err == nil {
			return fmt.Errorf("host resource visible: %s", path)
		}
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, os.ErrPermission) {
			return err
		}
	}
	for _, s := range os.Environ() {
		if forbiddenEnv(strings.SplitN(s, "=", 2)[0]) {
			return fmt.Errorf("credential environment present")
		}
	}
	for _, address := range []string{"1.1.1.1:443", "140.82.112.6:443"} {
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err == nil {
			conn.Close()
			return fmt.Errorf("external transport available: %s", address)
		}
	}
	// These are transport probes, not GitHub operation or authorization tests.
	if err = contracts("/spike/bin/advisory"); err != nil {
		return err
	}
	fmt.Println(`{"isolation":"passed","model_eval":"not run"}`)
	return nil
}
