// The local Spike runner is separate from the renodash CLI.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var revisions = map[string]string{"published-advisory": "2654fc92c0bb4c900b1d23dd5ae6af3caf4b6870", "govulncheck-discovery": "c27bcd21baf8196d2eabeddfa28427763048f415"}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: local-skill prepare|validate|isolate BUNDLE; contracts BINARY; probe; snapshot")
	}
	if args[0] == "probe" && len(args) == 1 {
		return probe()
	}
	if args[0] == "snapshot" && len(args) == 1 {
		return json.NewEncoder(os.Stdout).Encode(snapshot("18.9.0", "18.9.1"))
	}
	if len(args) != 2 {
		return fmt.Errorf("expected command and absolute bundle or binary path")
	}
	path, err := filepath.Abs(args[1])
	if err != nil {
		return err
	}
	switch args[0] {
	case "prepare":
		return prepare(path)
	case "validate":
		return validate(path)
	case "contracts":
		return contracts(path)
	case "isolate":
		return isolate(path)
	}
	return fmt.Errorf("unknown command %q", args[0])
}
func output(dir string, env []string, args ...string) ([]byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s: %w: %s", args[0], err, stderr.String())
	}
	return out, nil
}
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
func root() (string, error) {
	out, err := output("", nil, "git", "rev-parse", "--show-toplevel")
	return strings.TrimSpace(string(out)), err
}
func prepare(dst string) error {
	repo, err := root()
	if err != nil {
		return err
	}
	if err = os.Mkdir(dst, 0755); err != nil {
		return err
	}
	for name, rev := range revisions {
		prefix := "adapters/" + name + "/"
		files, err := output(repo, nil, "git", "ls-tree", "-rz", "--name-only", rev, "--", prefix)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return fmt.Errorf("missing adapter at %s; fetch the Spike branch", rev)
		}
		for _, file := range strings.Split(strings.TrimSuffix(string(files), "\x00"), "\x00") {
			if !filepath.IsLocal(file) {
				return fmt.Errorf("unsafe export path %q", file)
			}
			target := filepath.Join(dst, file)
			if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			data, err := output(repo, nil, "git", "show", rev+":"+file)
			if err != nil {
				return err
			}
			if err = os.WriteFile(target, data, 0644); err != nil {
				return err
			}
		}
	}
	runtime := filepath.Join(dst, "runtime")
	if err = os.MkdirAll(filepath.Join(runtime, "bin"), 0755); err != nil {
		return err
	}
	env := append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	if _, err = output(filepath.Join(dst, "adapters/published-advisory"), env, "go", "build", "-trimpath", "-buildvcs=false", "-o", filepath.Join(runtime, "bin/advisory"), "."); err != nil {
		return err
	}
	if _, err = output(repo, env, "go", "build", "-trimpath", "-buildvcs=false", "-o", filepath.Join(runtime, "bin/spike"), "./spikes/local-skill"); err != nil {
		return err
	}
	skill := filepath.Join(runtime, "renodash-local-advisory")
	if err = os.Mkdir(skill, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(repo, "examples/skills/renodash-local-advisory"))
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			return fmt.Errorf("unexpected Skill directory %s", e.Name())
		}
		if err = copyFile(filepath.Join(repo, "examples/skills/renodash-local-advisory", e.Name()), filepath.Join(skill, e.Name())); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(revisions, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(runtime, "provenance.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	if err = copyFile(filepath.Join(repo, "LICENSE"), filepath.Join(runtime, "LICENSE")); err != nil {
		return err
	}
	fmt.Println(dst)
	return nil
}
func validate(bundle string) error {
	repo, err := root()
	if err != nil {
		return err
	}
	skill := filepath.Join(repo, "examples/skills/renodash-local-advisory")
	data, err := os.ReadFile(filepath.Join(skill, "SKILL.md"))
	if err != nil {
		return err
	}
	// Resource checks only; the optional upstream validator checks metadata.
	for _, m := range regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`).FindAllStringSubmatch(string(data), -1) {
		target := m[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") {
			continue
		}
		if info, err := os.Stat(filepath.Join(skill, strings.Split(target, "#")[0])); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("missing linked file %s", target)
		}
	}
	for _, resource := range []string{"bin/advisory", "bin/spike", "provenance.json"} {
		info, err := os.Stat(filepath.Join(bundle, "runtime", resource))
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("missing resource %s", resource)
		}
	}
	if err = contracts(filepath.Join(bundle, "runtime/bin/advisory")); err != nil {
		return err
	}
	fmt.Println("Linked files, command resources, adapter contracts: PASS (metadata validator not run)")
	return nil
}
