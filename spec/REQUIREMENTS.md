# MyCal requirements

## Device login for API tokens

The shared user-visible contract is [MySuite device login](../../mysuite/spec/device-login.md).
The token CLI's `login` command reads no Basic credentials. It prints a short code
and approval URL, then polls while the user reviews and grants resources/lifetime
on a dedicated `device-approval.html` page. Requested scope and lifetime are
pre-filled and editable; a successful decision attempts to close the page, with
a completion message if closing is blocked. Manual code entry remains available
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

## API request limits

Create and update event requests cap color, recurrence-until and duration strings at 100 characters,
recurrence lists (`recurrence_by_day`, `recurrence_by_monthday`, `recurrence_by_month`, `exdates`, `rdates`)
at 5000 characters, and recurrence intervals at 999. The demo backend applies the same limits.
Token creation, device authorization and device consent accept at most 100 calendars/scopes. Raw iCalendar imports, including
single-event imports, accept at most 10 MiB (10485760 bytes).

These bounds are published in OpenAPI for API consumers, including the Android client. Requests above
the newly explicit 100-character and 100-item bounds are rejected; clients should stay within them.
