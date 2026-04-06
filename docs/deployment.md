# Deployment Guide

mycal is designed to run behind a TLS-terminating reverse proxy. This document describes the requirements that the reverse proxy **must** satisfy for the application to be secure.

---

## 1. Authentication

**mycal has no built-in access control beyond HTTP Basic Auth.** Without it, every endpoint — events, import, delete, iCalendar feed — is publicly accessible.

**Always** start mycal with `-basic-auth-file` pointing to a bcrypt htpasswd file:

```bash
# Create the htpasswd file (requires apache2-utils or httpd-tools)
htpasswd -Bc /etc/mycal/htpasswd admin

# Start the server
./mycal -basic-auth-file /etc/mycal/htpasswd -data /var/lib/mycal
```

Do **not** expose mycal on a network interface without authentication enabled.

---

## 2. TLS

mycal does not terminate TLS itself. The reverse proxy is responsible for HTTPS. Connections between the reverse proxy and mycal may use plain HTTP on localhost or a private network.

---

## 3. Reverse proxy requirements

### 3.1 X-Forwarded-Host — mandatory

mycal's CSRF middleware uses the `X-Forwarded-Host` header to determine the server's public-facing hostname, and rejects state-changing requests whose `Origin` or `Referer` does not match it. If the reverse proxy forwards a client-supplied `X-Forwarded-Host` unmodified, an attacker can set the header to match their own origin and bypass the CSRF check.

**The reverse proxy must always overwrite `X-Forwarded-Host` with the actual public hostname.** Never pass the client's value through.

### 3.2 Rate limiting — mandatory

mycal has no built-in rate limiting. Without it the API is susceptible to DoS via bulk event creation or repeated search queries. The reverse proxy must enforce a per-IP request rate limit.

---

## 4. Example configurations

### Nginx

```nginx
# Rate-limiting zone: 10 requests/second per IP, burst of 20
limit_req_zone $binary_remote_addr zone=mycal:10m rate=10r/s;

server {
    listen 443 ssl;
    server_name calendar.example.com;

    ssl_certificate     /etc/letsencrypt/live/calendar.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/calendar.example.com/privkey.pem;

    location / {
        # Rate limiting
        limit_req zone=mycal burst=20 nodelay;

        proxy_pass http://127.0.0.1:8080;

        # Always set X-Forwarded-Host to the actual public hostname.
        # This overwrites any value supplied by the client, preventing
        # CSRF-check bypass.
        proxy_set_header X-Forwarded-Host $host;

        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name calendar.example.com;
    return 301 https://$host$request_uri;
}
```

### Apache 2

Requires `mod_proxy`, `mod_proxy_http`, `mod_ratelimit`, `mod_ssl`, and `mod_headers`. Enable them with:

```bash
a2enmod proxy proxy_http ratelimit ssl headers
```

```apache
# Rate-limiting zone — must be outside any VirtualHost block
# (defined at global or server level via mod_ratelimit)

<VirtualHost *:443>
    ServerName calendar.example.com

    SSLEngine on
    SSLCertificateFile    /etc/letsencrypt/live/calendar.example.com/fullchain.pem
    SSLCertificateKeyFile /etc/letsencrypt/live/calendar.example.com/privkey.pem

    ProxyPreserveHost On
    ProxyPass        / http://127.0.0.1:8080/
    ProxyPassReverse / http://127.0.0.1:8080/

    # Always overwrite X-Forwarded-Host with the actual public hostname.
    # RequestHeader set runs after ProxyPass so it replaces any client-supplied value.
    RequestHeader set X-Forwarded-Host "calendar.example.com"

    RequestHeader set X-Forwarded-Proto "https"

    # Rate limiting: 10 requests/second per connection
    <Location />
        SetOutputFilter RATE_LIMIT
        SetEnv rate-limit 10
    </Location>
</VirtualHost>

# Redirect HTTP to HTTPS
<VirtualHost *:80>
    ServerName calendar.example.com
    Redirect permanent / https://calendar.example.com/
</VirtualHost>
```

> **Note:** Apache's `mod_ratelimit` limits the *response* throughput (bytes/sec), not the request rate. For true per-IP request-rate limiting use `mod_qos` (available as a package on most distributions: `apt install libapache2-mod-qos`) and add `QS_SrvMaxConnPerIP 10` to the VirtualHost block.

### Caddy

```caddy
calendar.example.com {
    # Rate limiting (requires caddy-ratelimit module)
    rate_limit {remote.ip} 10r/s

    reverse_proxy 127.0.0.1:8080 {
        header_up X-Forwarded-Host {host}
    }
}
```

> **Note:** The built-in Caddy distribution does not include a rate-limiting module. Build Caddy with `xcaddy build --with github.com/mholt/caddy-ratelimit`, or use Nginx if you prefer not to build a custom binary.

---

## 5. Systemd service example

```ini
[Unit]
Description=mycal
After=network.target

[Service]
Type=simple
User=mycal
Group=mycal
ExecStart=/usr/local/bin/mycal \
    -addr 127.0.0.1 \
    -port 8080 \
    -data /var/lib/mycal \
    -basic-auth-file /etc/mycal/htpasswd
Restart=on-failure
RestartSec=5

# Harden the service
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/mycal

[Install]
WantedBy=multi-user.target
```

Bind mycal to `127.0.0.1` (via `-addr 127.0.0.1`) so it is only reachable from the local machine, not directly from the internet.
