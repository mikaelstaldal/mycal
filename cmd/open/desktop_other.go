//go:build !linux && !windows && !darwin

package main

import (
	"errors"
	"os/exec"
)

const maxBrowserURLLength = 16 << 10

func browserCommand(address string) *exec.Cmd {
	return exec.Command("xdg-open", address)
}

func installDesktop(_ string, _ bool) error {
	return errors.New("desktop registration requires Linux, Windows or macOS")
}
