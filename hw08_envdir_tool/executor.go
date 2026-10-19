package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunCmd runs a command + arguments (cmd) with environment variables from env.
func RunCmd(cmd []string, env Environment) (returnCode int) {
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "envdir: no command specified")
		return 1
	}

	c := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec
	c.Env = buildEnv(env)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "envdir: failed to run %q: %v\n", cmd[0], err)
		return 1
	}

	return 0
}

// buildEnv merge process envs with files envs.
func buildEnv(env Environment) []string {
	result := make([]string, 0, len(os.Environ())+len(env))
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if _, overridden := env[name]; overridden {
			continue
		}
		result = append(result, kv)
	}

	for name, v := range env {
		if v.NeedRemove {
			continue
		}
		result = append(result, name+"="+v.Value)
	}

	return result
}
