# Desktop integration

MyCal can appear in **Open With** for `.ics` files, including calendar attachments
opened from a desktop mail client. It works with a local or remotely hosted MyCal
server and uses your normal browser login. Opening a file opens MyCal directly
in one browser tab. Review the file contents, choose a calendar, and confirm
before importing the event.

The server can run on Linux regardless of the client operating system. The
optional Go helper supports Linux, Windows, and macOS clients. All three use the
same browser confirmation flow and your normal browser login.

The server URL follows the same precedence as `mycal-cli` and `mycal-token`:
an explicit `-url` flag, then a nonempty `MYCAL_URL` environment variable, then
`http://127.0.0.1:8080`. For example:

```bash
export MYCAL_URL=https://example.com/mycal/
mycal-open meeting.ics
mycal-open -install
```

Installation records the resolved URL in the desktop registration, so it works
even when the desktop session does not inherit your shell environment. Changing
`MYCAL_URL` affects direct invocations; re-run installation to change the URL
used by a registered desktop handler. On macOS, remove the existing generated
app before reinstalling, as described below.

## Build the helper

Build just the helper on any supported client with Go installed:

```bash
go build -o mycal-open ./cmd/open
```

On Windows, use `go build -o mycal-open.exe ./cmd/open`. Building the Linux server
or its frontend is not required to compile this helper. From a development
checkout, `./build-desktop.sh` cross-builds helpers for Linux, Windows, and macOS,
for both Intel/AMD (`amd64`) and ARM (`arm64`), under `dist/desktop/`.

Platform implementations and their tests use Go's operating-system filename
suffixes, so each build includes only its matching platform code. The Windows
and macOS helpers and test executables can be cross-compiled on Linux, but their
tests, native installation, and file opening must run on those operating
systems. These helpers are not
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

An optional [AppArmor runtime profile](APPARMOR.md) restricts the Linux helper's
direct resource access. Complete desktop registration before loading it.

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

The helper reads one UTF-8 iCalendar file, removes an optional UTF-8 byte-order
mark, and opens the configured server URL with a base64url-encoded fragment:

```text
https://example.com/mycal/#import=<base64url-encoded-ics>
```

MyCal decodes and removes the fragment before rendering, then opens the single
**Import Event** confirmation dialog. It also accepts import fragments in an
already open tab. No event is created until you confirm; the existing single-event
API rejects files containing multiple events. Cancelling or reloading discards
the pending import. Bulk imports remain available through the normal file picker.

The helper starts the browser directly. The browser makes a normal top-level
GET and uses its existing login. The helper and web frontend must
both include this fragment protocol; update the embedded helper in an existing
macOS app by reinstalling it. If an authentication proxy drops fragments during
login, sign in first and reopen the file. Opener relationships and popup
permissions are no longer needed.

Fragments are not sent in HTTP requests or Referer headers. However, the initial
URL contains the event data and can be visible to local process inspection,
browser extensions, browser history/synchronization, or authentication pages
before MyCal removes it. Removing the fragment from the current history entry
does not guarantee erasure from those other places. Do not share the import URL.

To bound browser launch arguments, the complete URL is limited to **16 KiB on
Linux/macOS** and **2,000 bytes on Windows**, whose shell launcher has a tighter
URL limit. Encoding increases the file size by about one third; the server URL
also takes space. The helper reads at most 12 KiB before encoding, and rejects
an oversized URL with instructions to use MyCal's **Import Event** file picker.
The URL cap is the actual limit, not a guaranteed file-size allowance: Windows
typically fits about 1.4 KiB of event data, and Linux/macOS fit slightly less
than 12 KiB. A longer server URL reduces those allowances further.
Long descriptions or embedded attachments can exceed the limit even for one
event. No file is uploaded automatically as a fallback.

Remote servers require HTTPS. Plain HTTP is supported for loopback addresses.
On Linux, the helper is listed in **Open With**, rather than the application
launcher, because it needs a file.

Platform references: [Windows file registration](https://learn.microsoft.com/en-us/windows/win32/shell/fa-file-types),
[Windows Default apps settings](https://learn.microsoft.com/en-us/windows/apps/develop/launch/launch-settings),
[Windows launcher limits][windows-url-limits],
[Apple application bundles][apple-bundles],
and [choosing a default application in Finder](https://support.apple.com/en-ie/guide/mac-help/mh35597/27/mac/27).

[apple-bundles]: https://developer.apple.com/library/archive/documentation/CoreFoundation/Conceptual/CFBundles/BundleTypes/BundleTypes.html

[windows-url-limits]: https://devblogs.microsoft.com/oldnewthing/20031210-00/?p=41553
