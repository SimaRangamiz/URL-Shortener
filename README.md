# URL Shortener — Part 4

A lightweight, performant HTTP URL shortener service implemented in Go 1.22+ featuring both in-memory and persistent SQLite storage options.

## Features

- **URL Shortening:** Accepts long URLs and returns a deterministic short code and URL.
- **Redirection:** Redirects short codes (`302 Found`) to their original long URLs.
- **Link Metadata:** Retrieves detailed JSON metadata (`url` and `created_at` in RFC3339 format) for a short code without triggering a redirect.
- **Idempotency:** Re-submitting the same normalized URL always yields the exact same short code.
- **Durable Persistence:** Optional GORM + SQLite backend that automatically restores links across server restarts.
- **URL Normalization & Validation:** Ensures valid `http`/`https` schemes, trims whitespace, standardizes paths, and prevents server-side requests (SSRF).

## Project Structure

```text
URL-Shortener_part4/
├── cmd/
│   └── server/
│       └── main.go          # Application entrypoint & HTTP server configuration
├── internal/
│   ├── handler/             # HTTP handlers & Go 1.22+ ServeMux routing
│   ├── link/                # Domain models, URL validation & Base62 code generation
│   └── store/               # Storage interface, MemoryStore & DBStore implementations
├── DECISIONS.md             # Documented design decisions (Parts 1–4)
├── README.md                # Usage, instructions & coverage report
├── go.mod                   # Module declaration & dependencies
└── go.sum                   # Dependency checksums

How to RunCommand-Line FlagsFlagDefaultDescription-addr:8080Listen address for the HTTP server-basehttp://localhost:8080Base URL used when building short_url-storememoryStorage backend to use: memory or sqlite-db-pathlinks.dbSQLite database file path (when -store=sqlite)

1. Run with In-Memory Store
Bash
go run ./cmd/server -addr :8080 -base http://localhost:8080 -store memory
2. Run with Persistent SQLite Store
Bash
go run ./cmd/server -addr :8080 -base http://localhost:8080 -store sqlite -db-path links.db
Example Usage (curl)
Shorten a URL
Bash
curl -s -X POST http://localhost:8080/api/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url":"[https://go.dev/doc/](https://go.dev/doc/)"}'
Response (201 Created):

JSON
{
  "code": "a1B2c3",
  "short_url": "http://localhost:8080/a1B2c3"
}
Redirect to Original URL
Bash
curl -sI http://localhost:8080/a1B2c3
Response (302 Found):

HTTP
HTTP/1.1 302 Found
Location: [https://go.dev/doc/](https://go.dev/doc/)
Date: Fri, 09 Oct 2026 00:00:00 GMT
Inspect Link Metadata
Bash
curl -s http://localhost:8080/api/v1/links/a1B2c3
Response (200 OK):

JSON
{
  "url": "[https://go.dev/doc/](https://go.dev/doc/)",
  "created_at": "2026-10-09T00:00:00Z"
}
Running Tests & Coverage
To execute all unit tests, error mappings, race conditions, and persistence restart simulations:

Bash
# Run all unit tests
go test -v ./...

# Run race detection
go test -race ./...

# Generate coverage profile
go test -coverprofile="coverage.out" ./...

# Display total statement coverage
go tool cover -func="coverage.out" | Select-Object -Last 1   # PowerShell (Windows)
# go tool cover -func="coverage.out" | tail -n1              # Bash / zsh
Statement Coverage Output
Plaintext
total:                                           (statements)            71.4%