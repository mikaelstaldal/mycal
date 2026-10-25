package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

func browserCommand(address string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", address)
	case "darwin":
		return exec.Command("/usr/bin/open", address)
	default:
		return exec.Command("xdg-open", address)
	}
}

func installDesktop(base string, setDefault bool) error {
	switch runtime.GOOS {
	case "linux":
		return installLinux(base, setDefault)
	case "windows":
		return installWindows(base, setDefault)
	case "darwin":
		return installMac(base, setDefault)
	default:
		return fmt.Errorf("desktop registration is not supported on %s", runtime.GOOS)
	}
}
