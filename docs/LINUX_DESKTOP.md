# Linux desktop integration

MyCal can appear in **Open With** for `.ics` files, including calendar attachments
opened from a desktop mail client. It works with a local or remotely hosted MyCal
server and uses your normal browser login. Opening a file shows a **Continue to
MyCal** button. Click it to open the web interface, review the file contents,
choose a calendar, and confirm before importing any events.

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
the file; reopen it from the desktop to start again. The helper is listed in
**Open With**, rather than the application launcher, because it needs a file.

To remove the integration, choose another default calendar application in your
desktop settings, remove `mycal-open.desktop` from the applications directory,
and run `update-desktop-database` on that directory. You can then remove the
helper executable.
