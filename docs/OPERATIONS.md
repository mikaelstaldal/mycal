# mycal — Operations Guide

This guide covers production installation of mycal on a Linux server, including TLS termination via a reverse proxy (e.g. nginx) and systemd service management.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Install the Binary](#install-the-binary)
3. [Create a System User](#create-a-system-user)
4. [Set Up the Data Directory](#set-up-the-data-directory)
5. [Set Up Authentication](#set-up-authentication)
6. [Configure systemd](#configure-systemd)
7. [Configure a Reverse Proxy](#configure-a-reverse-proxy)
8. [First Login](#first-login)
9. [iCalendar Feed](#icalendar-feed)
10. [Exporting Data](#exporting-data)
11. [Upgrading](#upgrading)

---

## Prerequisites

- A Linux server.
- nginx (or another reverse proxy capable of TLS termination).
- A valid TLS certificate for your domain (e.g. from Let's Encrypt).
- Go 1.26+, `ogen`, `tsc`, and `openapi-typescript` if building from source; otherwise download a pre-built binary.

---

## Install the Binary

### Build from source

```bash
git clone https://github.com/mikaelstaldal/mycal.git
cd mycal
./build.sh -o /usr/local/bin
```

---

## Create a System User

Run mycal as a dedicated non-root user.

```bash
useradd --system --home-dir /var/lib/mycal --shell /usr/sbin/nologin mycal
```

---

## Set Up the Data Directory

```bash
mkdir -p /var/lib/mycal
chown mycal:mycal /var/lib/mycal
chmod 0700 /var/lib/mycal
```

mycal creates `mycal.sqlite` in the data directory on first startup and applies schema migrations automatically on each subsequent start. Starting the *server* against a database written by a newer mycal is refused rather than migrated — if you see "was written by a newer version of mycal", roll the binary forward rather than downgrading the data. `-export-ics` is deliberately not subject to that check: it opens the database read-only and cannot damage it, so an older binary can still get your data out.

A database from the other direction is refused too. Versions of mycal before schema versioning existed did not stamp `PRAGMA user_version`, and current builds no longer carry the code that recognised those databases column by column; one is refused on startup with a message naming a mycal build that can still migrate it. Run that build once against the data directory, then upgrade again. `-export-ics` is *not* a way out of this one — it reads columns such a database does not have — so the older binary is the only route.

---

## Set Up Authentication

mycal uses HTTP Basic Auth backed by an htpasswd file (bcrypt). Create the file as the `mycal` user:

```bash
htpasswd -Bc /etc/mycal/htpasswd myuser
```

The file is parsed strictly at startup: every non-blank line must be a `username:bcrypt-hash` pair (bcrypt only — `htpasswd -B`), usernames must be unique, and the file must not be empty. mycal refuses to start otherwise. A malformed, non-bcrypt or duplicate entry is reported with its file and line number; a file with no entries at all is reported by name, there being no line to point at.

Protect the file:

```bash
chown mycal:mycal /etc/mycal/htpasswd
chmod 0600 /etc/mycal/htpasswd
```

> **Important:** HTTP Basic Auth must only be used over HTTPS. Never expose mycal on a non-loopback interface without TLS. The reverse proxy (see below) provides TLS termination.

---

## Configure systemd

Create `/etc/systemd/system/mycal.service`:

```ini
[Unit]
Description=mycal calendar application
After=network.target

[Service]
Type=exec
User=mycal
Group=mycal

LoadCredential=basic-auth:/etc/mycal/htpasswd

ExecStart=/usr/local/bin/mycal \
    -data /var/lib/mycal \
    -addr 127.0.0.1 \
    -port 8080 \
    -public-url https://calendar.example.com \
    -https \
    -basic-auth-file ${CREDENTIALS_DIRECTORY}/basic-auth

Restart=on-failure
RestartSec=5

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
PrivateTmp=true
ReadWritePaths=/var/lib/mycal

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
systemctl daemon-reload
systemctl enable mycal
systemctl start mycal
systemctl status mycal
```

View logs:

```bash
journalctl -u mycal -f
```

---

## Configure a Reverse Proxy

mycal does not terminate TLS itself. Place it behind a reverse proxy.

Start mycal with `-public-url https://calendar.example.com` (and `-https` to enable `Strict-Transport-Security`). The CSRF middleware rejects state-changing requests whose `Origin` or `Referer` does not match the configured public URL.

### Rate limiting and authentication abuse

Configure request-rate limiting in the reverse proxy or an upstream firewall before requests reach mycal.
mycal has no built-in request throttling, authentication lockout, or cap on concurrent bcrypt comparisons.
With `-basic-auth-file`, each request carrying parseable Basic credentials can trigger an expensive bcrypt
comparison, including unknown usernames and wrong passwords. A flood can exhaust CPU, and repeated password
attempts are not limited by the application. Use strong passwords as well as external rate limits.

Apply per-client limits to all paths and methods, including GET requests, static assets, and failed authentication
attempts, rather than only writes or a login endpoint. Add an aggregate request limit sized for the server's CPU
capacity so requests from many IP addresses cannot bypass the per-client protection. Tune sustained rates and
bursts under load with bcrypt enabled, allowing normal page loads, calendar subscriptions, and API clients.
The nginx values below are starting points, not a capacity guarantee. The aggregate budget is shared by all
clients: when exhausted, it rejects requests even from clients below their individual limit. Size it for expected
concurrent users while preserving CPU headroom. The example allows bounded bursts immediately to avoid delaying
normal page loads; those bursts can still cause CPU spikes, so tune burst sizes as well as sustained rates.

Use the client address observed by the edge proxy for limiting and logging. If another trusted proxy sits in front,
configure client-IP handling to trust only that proxy; never accept arbitrary client-supplied forwarding headers.
For fail2ban-style blocking of repeated 401 responses, use the edge proxy's access log: behind a reverse proxy,
mycal's authentication log records the proxy's address as `RemoteAddr`. Failure-based blocking supplements
request limits; it does not prevent the bcrypt work needed to produce the initial 401 responses.

Keep the backend listener private so clients cannot bypass these controls (see [Firewall](#firewall)). A firewall
that only limits new TCP connections is insufficient on its own, since one persistent connection can carry many
HTTP requests. Use HTTP request limiting at the proxy or an HTTP-aware upstream firewall.

After deployment, verify in a controlled test that excessive requests with invalid Basic credentials are rejected
at the edge, that rejected requests do not reach mycal, and that legitimate clients remain usable. Monitor proxy
rejections, 401 responses, and backend CPU usage when tuning the limits.

### nginx

Create `/etc/nginx/sites-available/mycal`, included from the `http` context of `/etc/nginx/nginx.conf`.
The zone definitions must be outside the `server` blocks, as required by
[nginx's request limiting module](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).

```nginx
# Shared request-rate state; these directives belong in the http context.
limit_req_zone $binary_remote_addr zone=mycal_per_ip:10m rate=5r/s;
limit_req_zone $server_name zone=mycal_total:1m rate=10r/s;

server {
    listen 80;
    server_name calendar.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name calendar.example.com;

    ssl_certificate     /etc/letsencrypt/live/calendar.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/calendar.example.com/privkey.pem;

    # Modern TLS settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;

    # Apply before forwarding any request to mycal; tune rates and bursts for your CPU.
    limit_req zone=mycal_per_ip burst=20 nodelay;
    limit_req zone=mycal_total burst=40 nodelay;
    limit_req_status 429;

    location / {
        proxy_pass http://127.0.0.1:8080;

        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable and test:

```bash
ln -s /etc/nginx/sites-available/mycal /etc/nginx/sites-enabled/mycal
nginx -t
systemctl reload nginx
```

#### TLS certificate (Let's Encrypt)

```bash
certbot --nginx -d calendar.example.com
```

Certbot will modify the nginx config to handle certificate renewal automatically.

### Apache 2

Requires `mod_proxy`, `mod_proxy_http`, `mod_ssl`, and `mod_headers`. Enable them with:

```bash
a2enmod proxy proxy_http ssl headers
```

```apache
<VirtualHost *:443>
    ServerName calendar.example.com

    SSLEngine on
    SSLCertificateFile    /etc/letsencrypt/live/calendar.example.com/fullchain.pem
    SSLCertificateKeyFile /etc/letsencrypt/live/calendar.example.com/privkey.pem

    ProxyPreserveHost On
    ProxyPass        / http://127.0.0.1:8080/
    ProxyPassReverse / http://127.0.0.1:8080/

    RequestHeader set X-Forwarded-Proto "https"
</VirtualHost>

# Redirect HTTP to HTTPS
<VirtualHost *:80>
    ServerName calendar.example.com
    Redirect permanent / https://calendar.example.com/
</VirtualHost>
```

The Apache configuration above provides TLS and proxying only. Add per-client and aggregate HTTP request limits
using a suitable request-limiting module or an upstream HTTP-aware firewall before exposing it publicly.
Apache's [mod_ratelimit](https://httpd.apache.org/docs/2.4/mod/mod_ratelimit.html) limits response bandwidth,
not request rates, and does not protect bcrypt verification. A per-IP connection cap alone is also insufficient.

### Caddy

```caddy
calendar.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

The Caddy configuration above provides TLS and proxying only. Configure per-client and aggregate HTTP request
limits through a suitable third-party module or an upstream HTTP-aware firewall before exposing it publicly.
Alternatively, use the nginx request-limiting configuration above.

---

## First Login

Open `https://calendar.example.com` in your browser. Log in with the username and password you set in the htpasswd file.

The calendar opens directly to the current month. Use the navigation controls to switch between day, week, month, year, and schedule views.

---

## iCalendar Feed

Subscribe to your calendar from any app that supports iCalendar (Google Calendar, Apple Calendar, Thunderbird, etc.) using:

```
https://myuser:mypassword@calendar.example.com/calendar.ics
```

The feed includes all events and is regenerated on each request. Because Basic Auth credentials are embedded in the URL, treat this URL as a secret.

---

## Exporting Data

You can export all events to a `.ics` file while the server is running (the export opens the database read-only):

```bash
sudo -u mycal /usr/local/bin/mycal \
  -data /var/lib/mycal \
  -export-ics /tmp/mycal-backup.ics
```

---

## Upgrading

1. Build or download the new binary.
2. Stop the service:
   ```bash
   systemctl stop mycal
   ```
3. Replace the binary:
   ```bash
   install -o root -g root -m 0755 mycal-new /usr/local/bin/mycal
   ```
4. Start the service — schema migrations are applied automatically on startup:
   ```bash
   systemctl start mycal
   ```
5. Check the logs for any migration or startup errors:
   ```bash
   journalctl -u mycal -n 50
   ```

---

## Firewall

mycal binds to `127.0.0.1` by default and is never directly exposed to the internet. Ensure your firewall allows:

| Port | Protocol | Purpose                                  |
|------|----------|------------------------------------------|
| 80   | TCP      | HTTP → redirect to HTTPS (reverse-proxy) |
| 443  | TCP      | HTTPS (reverse-proxy → mycal)            |

The mycal process itself (port 8080) must not be reachable from outside the server.
