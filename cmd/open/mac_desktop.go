package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func appleScriptQuote(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`).Replace(value) + `"`
}

func macOpenScript(base string) string {
	return `on run
    display dialog "Open an iCalendar file with MyCal using Finder's Open With menu." buttons {"OK"} default button "OK"
end run

on open theFiles
    set helperPath to POSIX path of (path to resource "mycal-open")
    repeat with theFile in theFiles
        set filePath to POSIX path of (theFile as alias)
        try
            with timeout of 180 seconds
                do shell script quoted form of helperPath & " -url " & quoted form of ` + appleScriptQuote(base) + ` & " " & quoted form of filePath
            end timeout
        on error errorMessage
            display alert "Could not open iCalendar file" message errorMessage as warning
        end try
    end repeat
end open
`
}

const macDocumentTypes = `[{"CFBundleTypeName":"iCalendar File","CFBundleTypeRole":"Viewer","LSHandlerRank":"Alternate","LSItemContentTypes":["com.apple.ical.ics"],"CFBundleTypeExtensions":["ics","ical"],"CFBundleTypeMIMETypes":["text/calendar"]}]`

func installMac(base string, setDefault bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	applications := filepath.Join(home, "Applications")
	if err := os.MkdirAll(applications, 0755); err != nil {
		return err
	}
	destination := filepath.Join(applications, "MyCal.app")
	if _, err := os.Lstat(destination); err == nil {
		return errors.New("~/Applications/MyCal.app already exists; move it to Trash before reinstalling")
	} else if !os.IsNotExist(err) {
		return err
	}
	staging, err := os.MkdirTemp(applications, ".mycal-install-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	source := filepath.Join(staging, "MyCal.applescript")
	if err := os.WriteFile(source, []byte(macOpenScript(base)), 0600); err != nil {
		return err
	}
	bundle := filepath.Join(staging, "MyCal.app")
	if output, err := exec.Command("/usr/bin/osacompile", "-o", bundle, source).CombinedOutput(); err != nil {
		return fmt.Errorf("create Finder application: %w: %s", err, output)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(bundle, "Contents", "Resources", "mycal-open"), binary, 0755); err != nil {
		return err
	}
	plist := filepath.Join(bundle, "Contents", "Info.plist")
	for _, args := range [][]string{
		{"-replace", "CFBundleIdentifier", "-string", "com.mikaelstaldal.mycal.desktop", plist},
		{"-replace", "CFBundleName", "-string", "MyCal", plist},
		{"-replace", "CFBundleDocumentTypes", "-json", macDocumentTypes, plist},
	} {
		if output, err := exec.Command("/usr/bin/plutil", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("configure Finder application: %w: %s", err, output)
		}
	}
	// osacompile's original signature no longer covers the modified bundle.
	// Sign the locally generated app ad hoc; distribution signing is separate.
	if output, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", bundle).CombinedOutput(); err != nil {
		return fmt.Errorf("sign Finder application: %w: %s", err, output)
	}
	if err := os.Rename(bundle, destination); err != nil {
		return err
	}
	// Opening once lets Launch Services discover the application's document types.
	if err := exec.Command("/usr/bin/open", destination).Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "MyCal is installed at %s; open it in Finder to finish registration (%v).\n", destination, err)
	}
	if setDefault {
		_, _ = fmt.Fprintln(os.Stdout, "Select an .ics file in Finder, then Get Info > Open with > MyCal > Change All to choose the default.")
	}
	return nil
}
