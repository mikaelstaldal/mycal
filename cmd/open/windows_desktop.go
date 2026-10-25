package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// windowsQuote follows CommandLineToArgvW escaping; no command shell is used.
func windowsQuote(value string) string {
	var quoted strings.Builder
	quoted.WriteByte('"')
	slashes := 0
	for _, char := range value {
		if char == '\\' {
			slashes++
			continue
		}
		if char == '"' {
			quoted.WriteString(strings.Repeat("\\", slashes*2+1))
		} else {
			quoted.WriteString(strings.Repeat("\\", slashes))
		}
		slashes = 0
		quoted.WriteRune(char)
	}
	quoted.WriteString(strings.Repeat("\\", slashes*2))
	quoted.WriteByte('"')
	return quoted.String()
}

func windowsRegistration(executable, base string) [][]string {
	const classes = `HKCU\Software\Classes\`
	const progID = "MyCal.iCalendar"
	command := windowsQuote(executable) + " -url " + windowsQuote(base) + ` "%1"`
	name := executable[strings.LastIndexAny(executable, `/\`)+1:]
	application := classes + `Applications\` + name
	entries := [][]string{
		{classes + progID, "", "MyCal iCalendar File"},
		{classes + progID + `\shell\open\command`, "", command},
		{classes + `.ics\OpenWithProgids`, progID, ""},
		{classes + `.ical\OpenWithProgids`, progID, ""},
		{application, "FriendlyAppName", "MyCal"},
		{application + `\shell\open\command`, "", command},
		{application + `\SupportedTypes`, ".ics", ""},
		{application + `\SupportedTypes`, ".ical", ""},
		{`HKCU\Software\MyCal\Capabilities`, "ApplicationName", "MyCal"},
		{`HKCU\Software\MyCal\Capabilities`, "ApplicationDescription", "Open iCalendar files in the MyCal web interface"},
		{`HKCU\Software\MyCal\Capabilities\FileAssociations`, ".ics", progID},
		{`HKCU\Software\MyCal\Capabilities\FileAssociations`, ".ical", progID},
		{`HKCU\Software\RegisteredApplications`, "MyCal", `Software\MyCal\Capabilities`},
	}
	var commands [][]string
	for _, entry := range entries {
		args := []string{"add", entry[0], "/t", "REG_SZ", "/d", entry[2], "/f"}
		if entry[1] == "" {
			args = append(args, "/ve")
		} else {
			args = append(args, "/v", entry[1])
		}
		commands = append(commands, args)
	}
	return commands
}

func installWindows(base string, setDefault bool) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	for _, args := range windowsRegistration(executable, base) {
		if output, err := exec.Command("reg.exe", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("register MyCal for this Windows user: %w: %s", err, output)
		}
	}
	// Windows requires the user to choose defaults; never overwrite UserChoice.
	if setDefault {
		return browserCommand("ms-settings:defaultapps").Run()
	}
	return nil
}
