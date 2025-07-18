package main

import (
	"errors"
	"os"
	"os/exec"

	"github.com/justseemore/sshm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 130 {
			// 静默退出或自定义处理
			os.Exit(0)
		}
	}
}
