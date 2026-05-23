# plego — Plagger in Go

> **Plagger + Go = lightweight, fast, standalone RSS pipeline**

Plego is a full rewrite of [Plagger](https://github.com/plagger/plagger) (Pluggable RSS Aggregator) from Perl to Go.
Single binary, zero runtime deps, ~10 MB footprint. Designed for free-tier servers (Fly.io, Railway, Render).

## Why

|              | Plagger (Perl)     | plego (Go)          |
| ------------ | ------------------ | ------------------- |
| Image size   | ~200 MB (Docker)   | ~10 MB (static bin) |
| Startup      | ~3–5s (Perl init)  | <10ms               |
| Memory       | 50–100 MB RSS      | 5–15 MB RSS         |
| Parallelism  | Poor (Perl threads)| Native goroutines   |
| Deployment   | Docker required    | Single binary + cron|

**Heaviest Plagger user had 200+ feeds. plego handles 1000+ on a $0/mo instance.**

## Bounty

**1000 TTOKEN** to first implementation that passes the test suite.

Implementer keeps full copyright. Submit PR with working Go code.

## Pipeline

```
subscription (feed URL) → [filter ...] → publish (output)
```

Every phase is a Go interface. Built-in plugins are compiled in; external ones load via WASM or Unix pipe (TBD).

```
plego run -c config.yaml
plego api --addr :8080        # HTTP API mode
```

## Plugin interfaces

```go
// Subscription defines a feed source.
type Subscription interface {
  Name() string
  Feeds() ([]*Feed, error)
}

// Filter processes an entry stream.
type Filter interface {
  Name() string
  Filter(context.Context, *Entry) (*Entry, error) // return nil to drop
}

// Publish sends processed entries somewhere.
type Publish interface {
  Name() string
  Publish(context.Context, *Entry) error
}
```

### Built-in plugins

| Type         | Module           | Description                          |
| ------------ | ---------------- | ------------------------------------ |
| Subscription | Subscription::Config | Static feed list from YAML       |
| Subscription | Subscription::OPML   | OPML file import                |
| Filter       | Filter::Deduped      | GUID dedup (memory/boltdb)     |
| Filter       | Filter::Rule         | Title/content regex match      |
| Filter       | Filter::Truncate     | Trim body length               |
| Publish      | Publish::GAS         | POST to Google Apps Script API |
| Publish      | Publish::Stdout      | Print to stdout                |
| Publish      | Publish::File        | Append to local file           |
| Publish      | Publish::Webhook     | Generic HTTP POST              |

## Config (YAML)

Same shape as Plagger for easy migration:

```yaml
global:
  timezone: Asia/Tokyo

plugins:
  - module: Subscription::Config
    config:
      feed:
        - url: https://example.com/feed.xml
  - module: Filter::Deduped
  - module: Filter::Rule
    config:
      - match:
          title: Tech|Startup
  - module: Publish::Stdout
```

## API mode

Start HTTP server, trigger pipelines via REST:

```bash
plego api --addr :8080 --config config.yaml
```

| Method | Path             | Description                |
| ------ | ---------------- | -------------------------- |
| POST   | /run             | Execute pipeline once      |
| POST   | /run/:pipeline    | Run named pipeline section |
| GET    | /plugins         | List loaded plugins        |
| GET    | /health          | Health check               |

Designed for **cron + curl** on free-tier:

```cron
*/15 * * * * curl -X POST https://plego.example.com/run
```

## Project structure

```
plego/
├── cmd/
│   ├── plego/           # CLI entrypoint
│   └── plego-api/       # HTTP API server entrypoint
├── internal/
│   ├── feed/            # RSS/Atom fetch + parse
│   ├── filter/          # Filter implementations
│   ├── publish/         # Publisher implementations
│   ├── plugin/          # Plugin interface + registry
│   ├── config/          # YAML config loader
│   └── storage/         # Dedup state (memory, boltdb)
├── go.mod
├── go.sum
└── README.md
```

## Getting started (once implemented)

```bash
go install github.com/yourname/plego/cmd/plego@latest
plego run -c my-feeds.yaml
```

Or deploy to Fly.io:

```bash
fly launch --name plego
fly deploy
# set cron via fly.toml or external cron service
```

## Milestones

1. **Core**: feed fetch → filter chain → publish (CLI mode) — **300 TTOKEN**
2. **Dedup**: boltdb-backed Filter::Deduped — **100 TTOKEN**
3. **API**: HTTP server with /run endpoint — **200 TTOKEN**
4. **GAS publish**: POST to Google Apps Script Web App — **100 TTOKEN**
5. **OPML import**: Subscription::OPML — **100 TTOKEN**
6. **WASM plugin loader**: External plugins via WASM — **200 TTOKEN**

Total: **1000 TTOKEN**

## License

MIT
