# MyCal architecture

## Device login for API tokens

The shared user-visible contract is [MySuite device login](../../mysuite/spec/device-login.md).
The token CLI's `login` command reads no Basic credentials. It prints a short code
and approval URL, then polls while the user reviews and grants resources/lifetime
in Settings → API Tokens. Existing `create` and `revoke` commands remain available.

The server uses `go-server-common/devicegrant` with a versioned
`device_authorizations` table. POST `/api/v1/oauth/device_authorization` and
`/api/v1/oauth/token` bypass Basic auth but reject bearer credentials and retain
Host, CSRF and body-size protection. GET/POST `/api/v1/oauth/device/{user_code}`
require full access and approval POSTs require the normal CSRF policy. Consent
records a decision; polling mints through the existing token store and consumes
the device row in the same SQLite transaction. Secrets remain hash-only and
existing scope enforcement, listing and revocation are unchanged. Demo mode
does not support these endpoints. Requester IP is the socket peer; behind a
reverse proxy it is the proxy address, because untrusted forwarded headers are
ignored.
