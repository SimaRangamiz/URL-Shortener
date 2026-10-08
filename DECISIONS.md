# Design decisions

## Dependencies
- `gorm.io/gorm` & `gorm.io/driver/sqlite`: Used for Part 4 database persistence (allowed ORM + SQL driver as per requirements).
- No other external dependencies used (strictly stdlib for Part 1-3).

## Part 1 — MVP

### URL Normalization
To ensure consistent lookup and avoid duplicate entries for the same target, incoming URLs are normalized in `internal/link/validator.go`:
- Leading and trailing whitespace are trimmed.
- Scheme and host/domain parts are converted to lowercase.
- Trailing slashes at the end of the URL path are removed.
- Only non-empty URLs with `http` or `https` schemes are allowed.

### Same URL -> Same Code (Idempotency)
- Hashing is derived directly from the normalized URL using SHA-256 in `internal/link/generator.go`.
- Because SHA-256 is deterministic, identical URLs yield the exact same byte slice, which is mapped to a 6-character Base62 string.
- When `GetCodeOrShorten` is called with a URL that already exists in `MemoryStore`, the store checks if the generated code points to the same target URL and returns the existing code immediately.

### Code Generation & Collision Handling
- **Algorithm:** The normalized URL string is hashed via SHA-256. The first 6 bytes of the hash are converted into a Base62 string (`[0-9a-zA-Z]`) to produce a 6-character code.
- **Collision Handling:** If a generated code is already mapped to a *different* long URL, the generator appends a salt string (`#salt1`, `#salt2`, etc.) to the input URL and re-hashes. It retries up to 5 times until a unique slot is found.

### Mutex Choice & Locking Policy
- **Type:** `sync.RWMutex` is used in `MemoryStore` (`internal/store/memory.go`).
- **Read Operations:** `GetURL` and `GetLink` acquire a read lock (`RLock` / `RUnlock`) allowing concurrent lookup and redirect requests without blocking each other.
- **Write Operations:** `GetCodeOrShorten` acquires a full write lock (`Lock` / `Unlock`) to ensure safe atomic updates when registering new short codes.

### Package Layout
- `cmd/server`: Thin entrypoint parsing CLI flags (`-addr`, `-base`, `-store`, `-db-path`) and starting the HTTP server.
- `internal/link`: Domain models, URL validation, and Base62 code generation logic.
- `internal/store`: Storage interface, `MemoryStore` implementation using `sync.RWMutex`, and `DBStore` using GORM + SQLite.
- `internal/handler`: HTTP routing using Go 1.22+ `ServeMux`, request parsing, response encoding, and handlers for shorten, redirect, and metadata flows.

## Part 2

- **`Store` Interface Location & Methods:**
  - Defined the `Store` interface in `internal/store/store.go` to maintain a clear boundary between HTTP application logic and storage/domain implementations.
  - Interface methods include:
    - `GetCodeOrShorten(longURL string) (string, error)`: Handles idempotent link generation or lookup.
    - `GetURL(code string) (string, bool)`: Retrieves the long URL for redirects.
    - `GetLink(code string) (link.Link, error)`: Retrieves detailed link metadata (including creation timestamp).
  - Used an in-memory implementation (`MemoryStore`) for runtime operation and a mock/fake store (`FakeStore`) inside `internal/handler/handler_test.go` to test HTTP handlers independently.

- **Error Mapping & Status Codes:**
  - Defined explicit sentinel errors (`ErrNotFound` in `store` package and `ErrInvalidURL` in `link` package).
  - Errors are wrapped using `%w` when propagated from the store/domain layers.
  - Handlers inspect errors using `errors.Is` to map domain errors to specific HTTP responses:
    - `link.ErrInvalidURL` -> **400 Bad Request** with JSON error payload.
    - `store.ErrNotFound` -> **404 Not Found** with JSON error payload.
    - Malformed JSON input or empty `url` fields -> **400 Bad Request**.

- **Metadata Endpoint:**
  - Added `GET /api/v1/links/{code}` route to inspect stored links without triggering a redirect.
  - Returns `200 OK` with JSON containing original `url` and `created_at` in RFC3339 format upon success, or `404 Not Found` if the code does not exist.

- **Idempotency Persistence across Layers:**
  - Idempotency (same long URL returning the exact same short code) remains preserved through the `Store` abstraction layer and HTTP endpoints after the Part 2 refactor, as validated by `TestIdempotencyAfterRefactor`.

## Part 3

- **Locking choice (`sync.RWMutex`):**
  - In a URL shortener service, read operations (resolving/redirecting short codes and fetching metadata) are significantly more frequent than write operations (creating new short links).
  - Using `sync.RWMutex` allows multiple concurrent readers (`RLock`) to access the `links` map simultaneously without blocking each other.
  - Write operations (`GetCodeOrShorten`) acquire a full write lock (`Lock`), ensuring data consistency and race-free mutations when generating or persisting new links.

- **Timeout values and expected slow-client behavior:**
  - `ReadTimeout: 5 * time.Second`: Prevents slowloris-style attacks and frees up server resources if a client takes longer than 5 seconds to send the request headers and body.
  - `WriteTimeout: 10 * time.Second`: Ensures the server closes connections if response writing hangs or stalls beyond 10 seconds.
  - `IdleTimeout: 120 * time.Second`: Keeps keep-alive connections open for 2 minutes to reduce latency for subsequent requests while automatically reclaiming idle connections.
  - If a slow client exceeds these limits, the connection is abruptly closed by the Go HTTP server, protecting the service from connection exhaustion.

- **Eviction cap:**
  - An eviction mechanism (e.g., LRU or TTL-based eviction) was intentionally not implemented for the current in-memory store.
  - The dataset fits easily within memory for the benchmark scope, and adding eviction complexity would introduce extra locking overhead without immediate benefit before moving to persistent storage (Part 4).

## Part 4

- **Storage Choice:**
  - Selected GORM with SQLite as the relational persistent store (`DBStore`).
  - Implemented the `store.Store` interface, enabling seamless switching between `MemoryStore` and `DBStore` via configuration or CLI flags without changing handler logic.

- **Schema / Models and Migrations:**
  - Defined the `link.Link` struct with `Code` (primary key, `varchar(10)`), `OriginalURL` (indexed for quick idempotency lookups, `text`), and `CreatedAt` (`time.Time`).
  - Used GORM's `db.AutoMigrate(&link.Link{})` inside `NewDBStore` to automatically manage schema setup on application startup.

- **Startup Load:**
  - On startup, the database connections are established and auto-migrations run. Because links are persisted directly in SQLite table records, all existing shortenings and metadata persist across restarts without requiring an external dump file.

- **Crash Safety and Atomicity:**
  - Writes are executed synchronously using GORM's `s.db.Create(&newLink)`. The database transaction commits and persists the record to disk before returning the HTTP 201 Created response to the client.

- **How `created_at` is Stored:**
  - Handled via GORM's `autoCreateTime` tag on `CreatedAt`. SQLite stores timestamp values as standard UTC datetime/text strings, which are then formatted to RFC3339 in HTTP metadata responses.

- **Idempotency & Persistence:**
  - `DBStore` derives short codes deterministically from normalized long URLs. Upon receiving a shorten request, it directly queries the primary key (`code`) to check if the link already exists before inserting.
  - This guarantees that submitting the same normalized long URL after a service restart always yields the exact same short code without duplicate database writes or performance degradation.