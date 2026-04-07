# Known Issues

## Performance Issues

### Network & Assets

#### P18: All 30+ event columns fetched for every query
- **File:** `internal/repository/sqlite.go:159`
- `selectColumnsBase` fetches all columns including `latitude`, `longitude`, `categories`, `url`, `exdates`, `recurrence_by_month`, etc. on every query, even for list views that only display title and time.
