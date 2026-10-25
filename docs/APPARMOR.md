# AppArmor for the Linux desktop helper

[`deploy/apparmor/mycal-open`](../deploy/apparmor/mycal-open) is an optional
runtime profile for the Linux `mycal-open` client. It does not apply to the
MyCal server, Windows, or macOS.

## Scope

The profile grants read access to calendar files owned by the invoking user
under home directories, `/tmp`, `/var/tmp`, `/run/user`, `/media`, and `/mnt`.
The allowed suffixes are `.ics`, `.ical`, `.ICS`, and `.ICAL`. Symlinks are
checked against their resolved target paths. Files without those suffixes,
files owned by another account, or files elsewhere require a local rule.
The helper gets the Go runtime metadata, entropy, terminal access, and IPv4 TCP
permissions it needs, but no regular-file writes or installer commands.
Runtime metadata paths were derived from the Go 1.26 source, without a system
call trace. Other Go releases or cgroup mount layouts may require additional
read-only rules; repeat the enforced smoke test after upgrading the helper.

The profile uses the AppArmor 3.0 policy ABI and coarse IPv4 TCP mediation for
compatibility. It does **not** enforce a loopback-only network restriction:
`network inet stream` permits IPv4 TCP generally. The application's listener
still binds only to `127.0.0.1`; the browser makes the connection to the MyCal
server. No IPv6, UDP, DNS configuration access, or Linux capabilities are granted
to the helper.

`xdg-open` is a deliberate trust boundary. Its `rPUx` execution rule transitions
to a launcher profile if one exists, otherwise launches it unconfined while
scrubbing loader-sensitive environment variables. The launcher and browser do
not inherit the helper's file restrictions. This keeps the user's normal browser
and its own confinement working, but also means this profile cannot contain a
compromised helper that misuses the trusted launcher. It is a restriction on
the helper's direct resource access, rather than a complete exploit sandbox.

## Prepare the executable and desktop registration

Use a statically linked executable so no dynamic-loader or shared-library
permissions are needed:

```bash
CGO_ENABLED=0 go build -trimpath -tags netgo -o mycal-open ./cmd/open
install -Dm755 mycal-open ~/.local/bin/mycal-open
~/.local/bin/mycal-open -url https://example.com/mycal/ -install
```

The client binaries from `./build-desktop.sh` are also built with CGO disabled.
The profile automatically attaches to executables named `mycal-open` under
`/usr/bin`, `/usr/local/bin`, and `~/.local/bin`, using AppArmor's configured home
directory locations. Development checkout paths and the architecture-suffixed
cross-build filenames do not match. Adjust `@{mycal_open}` before loading the
profile if you use another installation path.

Complete `-install` and any `-set-default` operation **before** enabling the
profile. These commands write desktop registration data and run tools that the
runtime profile intentionally does not allow. To change the server URL later,
edit your own `mycal-open.desktop` file outside this profile, or remove the
profile temporarily while re-running desktop registration.

## Check and load the profile

On the Linux client with AppArmor enabled, check the policy without loading it
or using the parser cache:

```bash
apparmor_parser -Q -K deploy/apparmor/mycal-open
```

Install and load it as an enforcing profile:

```bash
sudo install -m 0644 deploy/apparmor/mycal-open /etc/apparmor.d/mycal-open
sudo apparmor_parser -r /etc/apparmor.d/mycal-open
sudo aa-status
```

The profile is prepared for host validation. This repository's authoring
sandbox denied execution of `apparmor_parser`, so kernel compilation and an
enforced desktop smoke test could not be performed there. The installed AppArmor
Python policy reader accepted the source; that is not a substitute for policy
compilation or kernel enforcement. Validate both on the target client before
relying on it. No AppArmor policy is installed or loaded by `mycal-open -install`.

Open a real `.ics` file using **Open With > MyCal**. Confirm that the helper page
and MyCal's import dialog open, then cancel and confirm the calendar is unchanged.
Open it again and confirm an import. Also try an invalid calendar file and
inspect kernel audit messages for unexpected denials:

```bash
sudo journalctl -k --since '5 minutes ago' | grep 'profile="mycal-open"'
```

Depending on the Go version and startup path, optional IPv6 capability probes
may generate `inet6` denials. These probes can fail gracefully while the IPv4
listener works; they do not require enabling IPv6 in this profile.

To check enforcement, try a readable calendar copied to a home-directory
filename without a permitted extension. Running the helper directly on that
path should fail with permission denied and produce a denial for `mycal-open`.

## Local adjustments and removal

Put extra rules in `/etc/apparmor.d/local/mycal-open`, then reload the profile.
For example, to allow a specific attachment directory whose files have no
extension:

```text
owner @{HOME}/.cache/my-mail-client/calendar-attachments/** r,
```

Custom XDG directories, shared files owned by another account, unusual home
directories, and network mounts may need additional path rules. Prefer specific
read-only paths over broad home-directory access. A different `xdg-open`
installation path also needs an explicit execution rule. The helper looks up
`xdg-open` through `PATH`: a custom wrapper earlier in `PATH`, such as
`~/.local/bin/xdg-open`, will be denied even if `/usr/bin/xdg-open` is present.
Use a `PATH` that selects the system launcher or explicitly allow a trusted
alternative in the local policy. If a launcher already has its own profile,
configure browser-launch permissions in that profile.

To unload the policy and prevent it being loaded on the next boot:

```bash
sudo apparmor_parser -R /etc/apparmor.d/mycal-open
sudo rm /etc/apparmor.d/mycal-open
```

Reference: [AppArmor policy language and execution transitions][apparmor-policy].

[apparmor-policy]: https://www.apparmor.net/man/4.0/apparmor.d/
