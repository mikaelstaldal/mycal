package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Desktop Exec uses its own quoting rules, independently of shell quoting.
func desktopQuote(value string) string {
	quoted := `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`, "%", "%%").Replace(value) + `"`
	// Desktop-entry string escaping is applied before Exec command parsing.
	return strings.ReplaceAll(quoted, `\`, `\\`)
}

func installLinux(base string, setDefault bool) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if !filepath.IsAbs(dataHome) {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	applications := filepath.Join(dataHome, "applications")
	if err := os.MkdirAll(applications, 0755); err != nil {
		return err
	}
	entry := "[Desktop Entry]\nType=Application\nName=MyCal\nComment=Open iCalendar files in MyCal\nNoDisplay=true\nTerminal=false\nCategories=Office;Calendar;\nMimeType=text/calendar;\nExec=" + desktopQuote(executable) + " -url " + desktopQuote(base) + " %f\n"
	if strings.ContainsAny(executable+base, "\r\n") {
		return errors.New("desktop entry values cannot contain line breaks")
	}
	if err := os.WriteFile(filepath.Join(applications, "mycal-open.desktop"), []byte(entry), 0644); err != nil {
		return err
	}
	if err := exec.Command("update-desktop-database", applications).Run(); err != nil {
		return fmt.Errorf("desktop entry written, but updating MIME associations failed: %w", err)
	}
	if setDefault {
		return exec.Command("xdg-mime", "default", "mycal-open.desktop", "text/calendar").Run()
	}
	return nil
}
