# Credits

mycal is built on the work of many open source projects. This file lists the third-party
libraries, assets and data sources used, together with their licenses.

mycal itself is Copyright 2026 Mikael Ståldal, licensed under the Apache License 2.0 (see [LICENSE](LICENSE)).

## Backend (Go)

### Direct dependencies

| Library | Version | License |
|---|---|---|
| [github.com/go-faster/errors](https://github.com/go-faster/errors) | v0.7.1 | BSD-3-Clause |
| [github.com/go-faster/jx](https://github.com/go-faster/jx) | v1.2.0 | MIT |
| [github.com/microcosm-cc/bluemonday](https://github.com/microcosm-cc/bluemonday) | v1.0.27 | BSD-3-Clause |
| [github.com/mikaelstaldal/go-server-common](https://github.com/mikaelstaldal/go-server-common) | v1.8.0 | Apache-2.0 |
| [github.com/ogen-go/ogen](https://github.com/ogen-go/ogen) | v1.22.0 | Apache-2.0 |
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | v1.52.0 | BSD-3-Clause |

`modernc.org/sqlite` is a pure-Go translation of [SQLite](https://sqlite.org/), which is in the public domain.

### Transitive dependencies linked into the binary

| Library | Version | License |
|---|---|---|
| [github.com/aymerick/douceur](https://github.com/aymerick/douceur) | v0.2.0 | MIT |
| [github.com/dlclark/regexp2](https://github.com/dlclark/regexp2) | v1.12.0 | MIT |
| [github.com/dustin/go-humanize](https://github.com/dustin/go-humanize) | v1.0.1 | MIT |
| [github.com/fatih/color](https://github.com/fatih/color) | v1.19.0 | MIT |
| [github.com/ghodss/yaml](https://github.com/ghodss/yaml) | v1.0.0 | MIT and BSD-3-Clause |
| [github.com/go-faster/yaml](https://github.com/go-faster/yaml) | v0.4.6 | Apache-2.0 or MIT |
| [github.com/google/uuid](https://github.com/google/uuid) | v1.6.0 | BSD-3-Clause |
| [github.com/gorilla/css](https://github.com/gorilla/css) | v1.0.1 | BSD-3-Clause |
| [github.com/mattn/go-colorable](https://github.com/mattn/go-colorable) | v0.1.14 | MIT |
| [github.com/mattn/go-isatty](https://github.com/mattn/go-isatty) | v0.0.22 | MIT |
| [github.com/remyoudompheng/bigfft](https://github.com/remyoudompheng/bigfft) | v0.0.0-20230129092748 | BSD-3-Clause |
| [github.com/segmentio/asm](https://github.com/segmentio/asm) | v1.2.1 | MIT |
| [github.com/shopspring/decimal](https://github.com/shopspring/decimal) | v1.4.0 | MIT |
| [go.uber.org/multierr](https://github.com/uber-go/multierr) | v1.11.0 | MIT |
| [go.uber.org/zap](https://github.com/uber-go/zap) | v1.28.0 | MIT |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | v0.53.0 | BSD-3-Clause |
| [golang.org/x/exp](https://pkg.go.dev/golang.org/x/exp) | v0.0.0-20260312153236 | BSD-3-Clause |
| [golang.org/x/net](https://pkg.go.dev/golang.org/x/net) | v0.56.0 | BSD-3-Clause |
| [golang.org/x/sync](https://pkg.go.dev/golang.org/x/sync) | v0.21.0 | BSD-3-Clause |
| [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) | v0.46.0 | BSD-3-Clause |
| [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) | v0.38.0 | BSD-3-Clause |
| [gopkg.in/yaml.v2](https://github.com/go-yaml/yaml) | v2.4.0 | Apache-2.0 |
| [modernc.org/libc](https://pkg.go.dev/modernc.org/libc) | v1.72.3 | BSD-3-Clause |
| [modernc.org/mathutil](https://pkg.go.dev/modernc.org/mathutil) | v1.7.1 | BSD-3-Clause |
| [modernc.org/memory](https://pkg.go.dev/modernc.org/memory) | v1.11.0 | BSD-3-Clause |

mycal is written in [Go](https://go.dev/), licensed under BSD-3-Clause. The Go standard
library is statically linked into the binary.

## Frontend (JavaScript), vendored in `web/static/third_party/`

These files are served to the browser and embedded in the mycal binary.

| Library | Version | License |
|---|---|---|
| [Preact](https://preactjs.com/) (`preact`, `hooks`, `jsx-runtime`) | 10.29.7 | MIT |
| [Quill](https://quilljs.com/) rich text editor (JS + Snow theme CSS) | 2.0.3 | BSD-3-Clause |
| [Leaflet](https://leafletjs.com/) map library (JS + CSS + marker images) | 1.9.4 | BSD-2-Clause |

Quill's copyright notice: Copyright (c) 2017-2024, Slab; Copyright (c) 2014, Jason Chen;
Copyright (c) 2013, salesforce.com.

`quill-2.0.3.js` is Quill's prebuilt bundle, which additionally inlines the following
libraries:

| Library | Version | License |
|---|---|---|
| [parchment](https://github.com/quilljs/parchment) | 3.0.0 | BSD-3-Clause |
| [quill-delta](https://github.com/quilljs/delta) | 5.1.0 | MIT |
| [eventemitter3](https://github.com/primus/eventemitter3) | 5.0.4 | MIT |
| [fast-diff](https://github.com/jhchen/fast-diff) | 1.3.0 | Apache-2.0 |
| [lodash-es](https://lodash.com/) | 4.18.1 | MIT |
| [lodash.clonedeep](https://lodash.com/) | 4.5.0 | MIT |
| [lodash.isequal](https://lodash.com/) | 4.5.0 | MIT |

### Type declarations only

The packages under `web/ts/third_party/node_modules/` supply TypeScript type declarations at
compile time and are not shipped. `web/ts/third_party/leaflet.d.ts` and `web/ts/third_party/quill.d.ts`
are hand-written ambient declarations original to this project. `web/ts/third_party/preact/`
contains Preact's own `.d.ts` files (MIT).

## Map data and external services

The map picker is optional and off unless a provider is configured in settings.

- **[OpenStreetMap](https://www.openstreetmap.org/)** — map tiles are fetched from
  `tile.openstreetmap.org`. Map data © OpenStreetMap contributors, available under the
  [Open Database License (ODbL)](https://www.openstreetmap.org/copyright); the rendered
  tiles are available under [CC BY-SA 2.0](https://creativecommons.org/licenses/by-sa/2.0/).
- **[Google Maps JavaScript API](https://developers.google.com/maps/documentation/javascript)** —
  loaded from `maps.googleapis.com` only when the user supplies an API key, and subject to
  the [Google Maps Platform Terms of Service](https://cloud.google.com/maps-platform/terms).

## Standards

- **iCalendar** import, export and feeds implement [RFC 5545](https://www.rfc-editor.org/rfc/rfc5545)
  (and related RFCs). The implementation in `internal/ical/` is original to this project
  and uses no external library.
- The REST API is described with [OpenAPI](https://www.openapis.org/).

## Build and development tools

These are required to build or test mycal but are not distributed with it.

| Tool | License |
|---|---|
| [ogen](https://ogen.dev/) — Go server stub generation | Apache-2.0 |
| [TypeScript](https://www.typescriptlang.org/) (`tsc`) | Apache-2.0 |
| [openapi-typescript](https://github.com/openapi-ts/openapi-typescript) | MIT |
| [golangci-lint](https://golangci-lint.run/) | GPL-3.0 |
| [Playwright](https://playwright.dev/) — end-to-end tests | Apache-2.0 |

### Go test dependencies

| Library | Version | License |
|---|---|---|
| [github.com/stretchr/testify](https://github.com/stretchr/testify) | v1.11.1 | MIT |
| [github.com/davecgh/go-spew](https://github.com/davecgh/go-spew) | v1.1.1 | ISC |
| [github.com/pmezard/go-difflib](https://github.com/pmezard/go-difflib) | v1.0.0 | BSD-3-Clause |
| [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) | v3.0.1 | MIT |
