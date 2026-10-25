# Desktop integration

MyCal can appear in **Open With** for `.ics` files, including calendar attachments
opened from a desktop mail client. It works with a local or remotely hosted MyCal
server and uses your normal browser login. Opening a file shows a **Continue to
MyCal** button. Click it to open the web interface, review the file contents,
choose a calendar, and confirm before importing any events.

The server can run on Linux regardless of the client operating system. The
optional Go helper supports Linux, Windows, and macOS clients. All three use the
same browser confirmation flow and your normal browser login.

## Build the helper

Build just the helper on any supported client with Go installed:

```bash
go build -o mycal-open ./cmd/open
```

On Windows, use `go build -o mycal-open.exe ./cmd/open`. Building the Linux server
or its frontend is not required to compile this helper. From a development
checkout, `./build-desktop.sh` cross-builds helpers for Linux, Windows, and macOS,
for both Intel/AMD (`amd64`) and ARM (`arm64`), under `dist/desktop/`.

The Windows and macOS helpers can be cross-compiled and their generated
registration data tested on Linux, but their native installation and file
opening must also be verified on those operating systems. These helpers are not
distributed with a publisher certificate or Apple notarization. The macOS
installer signs the app it generates locally with an ad hoc signature.

## Linux

Build with `./build.sh`, then keep the resulting `mycal-open` executable at a
permanent location, for example `~/.local/bin/mycal-open`. The desktop entry
records its absolute path. The Linux desktop utilities `xdg-open`, `xdg-mime`
(when setting the default), and `update-desktop-database` must be available.

```bash
install -Dm755 mycal-open ~/.local/bin/mycal-open
```

Register the helper with your MyCal URL:

```bash
~/.local/bin/mycal-open -url https://example.com/mycal/ -install
```

To also make MyCal the default application for iCalendar files:

```bash
~/.local/bin/mycal-open -url https://example.com/mycal/ -install -set-default
```

The registration writes `mycal-open.desktop` under
`${XDG_DATA_HOME:-~/.local/share}/applications`. Re-run the command to change the
server URL. You can also invoke the helper directly:

```bash
mycal-open -url http://localhost:8080/ meeting.ics
```

To remove the Linux integration, choose another default calendar application in
your desktop settings, remove `mycal-open.desktop` from the applications
directory, and run `update-desktop-database` on that directory. You can then remove
the helper executable.

## Windows

Keep `mycal-open.exe` in a permanent location, for example
`%LOCALAPPDATA%\MyCal\mycal-open.exe`, then run from PowerShell:

```powershell
& "$env:LOCALAPPDATA\MyCal\mycal-open.exe" -url https://example.com/mycal/ -install
```

This registers MyCal under your own Windows account, without administrator
privileges, for `.ics` and `.ical` files. Right-click a file and select
**Open with > MyCal**. The registered command records the executable's absolute
path; re-run installation if you move it or change the server URL.

To open Windows' Default apps settings after registration, add `-set-default`:

```powershell
& "$env:LOCALAPPDATA\MyCal\mycal-open.exe" -url https://example.com/mycal/ -install -set-default
```

Choose MyCal for `.ics` in those settings. Windows requires you to choose the
default; the helper does not overwrite the protected `UserChoice` registry key.
Depending on the desktop configuration, a console window may appear briefly
while the helper opens your browser.

To uninstall, first choose another default application. In Registry Editor,
remove the `MyCal.iCalendar` subkey under `HKEY_CURRENT_USER\Software\Classes`,
the `MyCal.iCalendar` values under `.ics\OpenWithProgids` and
`.ical\OpenWithProgids` there, the `HKEY_CURRENT_USER\Software\MyCal` key,
and the `MyCal` value in `HKEY_CURRENT_USER\Software\RegisteredApplications`.
Also remove `HKEY_CURRENT_USER\Software\Classes\Applications\mycal-open.exe`
(using your executable's actual filename). Then remove the executable. Preserve
other applications' entries.

## macOS

Run the appropriate `darwin` helper (ARM for Apple Silicon, AMD64 for Intel):

```bash
chmod +x ./mycal-open-darwin-arm64
./mycal-open-darwin-arm64 -url https://example.com/mycal/ -install
```

Installation uses macOS' built-in `osacompile`, `plutil`, and `codesign` tools to
create `~/Applications/MyCal.app`, containing a copy of the helper and the
configured server URL. Its AppleScript `open` handler receives files from Finder
and passes their paths to the helper. No Xcode project or Go installation is
needed when using a prebuilt helper. The installer opens the application once
so macOS can discover its document types.

In Finder, select an `.ics` file and use **Open With > MyCal**. If it is not yet
listed, select **Other** and choose `~/Applications/MyCal.app`. To make it the
default, select an `.ics` file, choose **Get Info > Open with > MyCal > Change
All**. Adding `-set-default` to the installation command prints these Finder
instructions; it does not change the default automatically.

You can move the generated app because its helper path is relative to the
bundle. To change the configured URL or update its embedded helper, move the
existing app to Trash and run installation again. The installer refuses to
overwrite an existing `~/Applications/MyCal.app`. To uninstall, choose another
default application and move the MyCal app to Trash.

## Browser handoff

Remote servers require HTTPS. Plain HTTP is supported for loopback addresses.
Both the server and helper must include desktop integration support. MyCal opens
with a normal top-level GET, so existing browser login cookies work normally.
If your authentication proxy replaces URL fragments or severs popup opener
relationships during login, sign in to MyCal first and then reopen the file.
The MyCal page must retain its opener relationship; a proxy setting
`Cross-Origin-Opener-Policy: same-origin` on MyCal prevents this integration.

The helper briefly serves a random, loopback-only URL to your browser. Clicking
**Continue to MyCal** opens MyCal in another tab, then transfers the file through
`postMessage`. Both pages check the other page's origin and window identity, and
the handoff also requires a matching random nonce. Only the helper address and
nonce appear in the navigation fragment; file contents never appear in a URL.
The import waits in memory until you confirm. The final import uses the existing
authenticated, CSRF-protected import API. No API token or local copy of your
server password is needed, and no server authentication or CSRF exception is
required. Once the file is sent, you can close the helper tab.

Desktop opening supports UTF-8 iCalendar files up to 10 MiB, including files with
a UTF-8 byte-order mark. Cancelling the dialog or reloading the MyCal tab discards
the file; reopen it from the desktop to start again. On Linux, the helper is
listed in **Open With**, rather than the application launcher, because it needs
a file.

Platform references: [Windows file registration](https://learn.microsoft.com/en-us/windows/win32/shell/fa-file-types),
[Windows Default apps settings](https://learn.microsoft.com/en-us/windows/apps/develop/launch/launch-settings),
[Apple application bundles][apple-bundles],
and [choosing a default application in Finder](https://support.apple.com/en-ie/guide/mac-help/mh35597/27/mac/27).

[apple-bundles]: https://developer.apple.com/library/archive/documentation/CoreFoundation/Conceptual/CFBundles/BundleTypes/BundleTypes.html
