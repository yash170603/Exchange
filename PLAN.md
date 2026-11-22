# Exchange: the 1-year plan

**What:** a small stock exchange. Traders connect, send buy and sell orders, the exchange
matches them fairly, broadcasts prices, saves trades, and survives crashes.

**Why:** two goals, in this order:
1. **Write idiomatic Go by hand.** Syntax, the standard library, and the habits of good Go code,
   built into muscle memory by typing every line yourself.
2. **Learn systems engineering.** Concurrency, networking, timeouts, rate limiting, databases,
   crash recovery, performance, replication. Each module is one of these skills.

The exchange is just the excuse. **You don't have to invent the business logic.** Claude hands
you the design of every function, including the rules it follows. Your job is to turn that
design into clean, tested Go and to understand *why* it's built that way.

**Budget:** ~1 hour a day, ~5 hours a week. Week counts are estimates. **Keep the order, not the dates.**

---

## How it works: function cards

Every module is a list of **files**, and every file is a list of **functions**. For each
function, Claude writes a **card** in `docs/hld/MX.md` before you start:

```text
### F3 · Submit                                   [ ] not started
File:       orderbook/book.go
Signature:  func (b *OrderBook) Submit(o *Order) ([]Trade, error)
Does:       plain-English steps, including the business rules
Go to use:  syntax and packages this function practises (e.g. slices.Insert, errors.Is)
System idea: the engineering concept behind it (e.g. why seq beats timestamps)
Tests:      the cases your test must cover
Watch out:  common mistakes
```

The loop for each card:

1. **Read the card.** Ask about anything unclear *before* typing.
2. **Type it yourself.** No copy-paste, no AI autocomplete in the editor.
3. **Write its test**, then `go test -race ./...` and `go vet ./...`.
4. **Ask Claude to review.** Fix what comes back. Claude points at lines, it doesn't rewrite.
5. **Commit** (one card = one commit) and Claude ticks the card here and in `docs/hld/MX.md`.

Stuck? Claude climbs the hint ladder in `CLAUDE.md`: question → concept → steps → small snippet.

At the end of each module: Claude quizzes you, and you write the short deliverable listed.

Card status: `[ ]` not started · `[~]` written, needs review/fixes · `[x]` reviewed and committed

---

## The finished system

```text
  trader programs (TCP)                          watchers (TCP/UDP)
        │ orders                                        ▲ live prices
        ▼                                               │
  ┌──────────────┐  M4 gateway                  ┌───────┴────────┐
  │   Gateway    │  M6 risk + rate limits       │ Market data    │ M5
  └──────┬───────┘                              │ feed           │
         │                                      └───────▲────────┘
         ▼                                              │
  ┌──────────────┐     ┌───────────────────┐            │
  │   Router     │────►│ Engine: RELIANCE  │ M1 + M2    │
  │  (by stock)  │────►│ Engine: TCS       │ one goroutine
  │              │────►│ Engine: INFY      │ per stock  │
  └──────────────┘     └─────────┬─────────┘            │
                                 │                      │
                    M9 journal ◄─┤ (written before      │
                    (crash       │  processing)         │
                     recovery)   ▼                      │
                        ┌─────────────────┐             │
                        │    Fan-out      │─────────────┘
                        └────────┬────────┘
                                 ▼
                        ┌─────────────────┐
                        │ Post-trade pool │ M3
                        └────────┬────────┘
                                 ▼
                           Postgres  M8

  Across everything: M7 logs, metrics, dashboards · M10 performance
  Later: M11 engine rewritten in Rust · M12 backup exchange that takes over on crash
```

## Folder layout

```text
exchange/
├── CLAUDE.md, PLAN.md, README.md, Makefile, go.mod
├── cmd/
│   ├── exchange/      ← main program: wires everything together
│   ├── trader/        ← fake trader / load generator (M2, M4)
│   └── watcher/       ← market data viewer (M5)
├── orderbook/  M1     ├── engine/   M2     ├── posttrade/ M3
├── gateway/    M4     ├── marketdata/ M5   ├── risk/      M6
├── metrics/    M7     ├── store/    M8     ├── journal/   M9
├── docs/
│   ├── hld/           ← function cards, one file per module (Claude writes these)
│   ├── adr/           ← decision records
│   └── notes/         ← your own notes
└── rust/       M11
```

Create each folder when its module starts, not before.

**Signatures for later modules are a first sketch.** Claude refines them into full cards
when the module starts, based on how the earlier code actually turned out.

---

## Module 1: Order book · 3 weeks · `docs/hld/M1.md`

**Goal:** one stock's order book: orders go in, trades come out. Pure logic, no goroutines.

**Go you'll learn:** structs, methods, pointer vs value receivers, `iota` enums, the `Stringer`
interface, slices (`append`, `slices.Insert`, `slices.Delete`, `slices.BinarySearchFunc`), maps,
errors (`errors.New`, `fmt.Errorf` with `%w`, `errors.Is`), the comma-ok idiom, table-driven tests,
`t.Run`, test helpers, example tests, fuzz tests.

**Systems you'll learn:** why money is never a float, why a counter beats a timestamp for
ordering, keeping data sorted vs sorting on every read, invariants a test can check.

| # | File | Function | Status |
|---|---|---|---|
| F0 | repo | Housekeeping: `.gitignore`, untrack the `exchange` binary | [ ] |
| F1 | `orderbook/order.go` | Types: `OrderType`, `OrderStatus`, `Order`, `Trade`, `OrderBook` | [x] |
| F2 | `orderbook/order.go` | `String()` for `OrderType` and `OrderStatus` | [ ] |
| F3 | `orderbook/errors.go` | Sentinel errors: `ErrInvalidQty`, `ErrInvalidPrice`, … | [ ] |
| F4 | `orderbook/book.go` | `NewOrderBook(entity string) *OrderBook` | [~] |
| F5 | `orderbook/book.go` | `(b *OrderBook) validate(o *Order) error` | [ ] |
| F6 | `orderbook/book.go` | `(b *OrderBook) rest(o *Order)`: insert keeping best-first | [ ] |
| F7 | `orderbook/book.go` | `(b *OrderBook) BestBid()` / `BestAsk()` with comma-ok | [ ] |
| F8 | `orderbook/book.go` | `(b *OrderBook) Submit(o *Order) ([]Trade, error)`: no matching yet | [~] |
| F9 | `orderbook/book.go` | `(b *OrderBook) match(o *Order) []Trade`: full, partial, sweep | [ ] |
| F10 | `orderbook/book.go` | `(b *OrderBook) Cancel(id int64) error` + `byID` map | [ ] |
| F11 | `orderbook/order.go` + `book.go` | Market orders: `Kind` (`Limit`, `Market`) | [ ] |
| F12 | `orderbook/book_test.go` | Table-driven tests for F4–F11 (grows with each card) | [ ] |
| F13 | `orderbook/example_test.go` | `ExampleOrderBook_Submit`: runnable documentation | [ ] |
| F14 | `orderbook/fuzz_test.go` | `FuzzSubmit`: random orders, book is never "crossed" | [ ] |

**Deliverable:** `docs/notes/M1.md`: the matching rules in your own words + one worked example.
**Done when:** all cards `[x]`, `go test -race ./...` green, you can trace a sweep by hand.

### After M1: tooling, one sitting

| # | What | Learn | Status |
|---|---|---|---|
| T1 | `Makefile`: `test`, `vet`, `run`, `fmt` | make basics, `.PHONY` | [ ] |
| T2 | `.github/workflows/ci.yml`: vet + `test -race` on every push | what CI is and why | [ ] |
| T3 | `README.md`: what it is, how to run it | writing for a stranger | [ ] |

---

## Module 2: Matching engine · 3 weeks · `docs/hld/M2.md`

**Goal:** many stocks at once, one goroutine per stock, clean shutdown.

**Go you'll learn:** goroutines, buffered vs unbuffered channels, directional channel types
(`<-chan`, `chan<-`), `select`, closing channels, `context.Context`, `os/signal` +
`signal.NotifyContext`, `errgroup`, type switches, the `flag` package, `math/rand/v2`, `time`.

**Systems you'll learn:** single-writer design (no locks needed), backpressure (wait vs reject when
a queue is full), graceful shutdown (stop intake → drain → exit), goroutine leaks, the race detector.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `engine/messages.go` | `Command` (submit/cancel) and `Event` (ack, reject, trade) types | [ ] |
| F2 | `engine/engine.go` | `New(entity string, in <-chan Command, out chan<- Event) *Engine` | [ ] |
| F3 | `engine/engine.go` | `(e *Engine) Run(ctx context.Context) error`: the per-stock loop | [ ] |
| F4 | `engine/router.go` | `NewRouter(entities []string, queueSize int) *Router` | [ ] |
| F5 | `engine/router.go` | `(r *Router) Route(ctx, cmd Command) error`: `ErrUnknownEntity`, `ErrBusy` | [ ] |
| F6 | `engine/router.go` | `(r *Router) Run(ctx) error`: start every engine with `errgroup` | [ ] |
| F7 | `cmd/exchange/main.go` | Wire router + engines, shut down on Ctrl+C | [ ] |
| F8 | `cmd/trader/main.go` | Flags (`-n`, `-stocks`), N random orders, print orders/sec | [ ] |
| F9 | `engine/*_test.go` | Tests: ordering kept, no leaks, shutdown drains the queue | [ ] |

**Deliverable:** `docs/adr/0001-one-goroutine-per-stock.md`
**Done when:** 100k orders across 10 stocks, `-race` clean; Ctrl+C loses no queued order.

---

## Module 3: Post-trade worker pool · 3 weeks · `docs/hld/M3.md`

**Goal:** do slow work after a trade (confirmations) with a worker pool: timeouts, retries, no duplicates.

**Go you'll learn:** your first real **interface** (`Handler`), `sync.WaitGroup`, `sync.Mutex`,
`context.WithTimeout`, functional options (`WithWorkers(n)`), custom error types, `testing/synctest`
(testing time-based code without real sleeps).

**Systems you'll learn:** timeouts only work if the slow code checks `ctx.Done()`, exponential
backoff + jitter, at-least-once delivery + idempotency, dead letter queues.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `posttrade/job.go` | `Job`, `Handler` interface, `Result` | [ ] |
| F2 | `posttrade/pool.go` | `NewPool(h Handler, opts ...Option) *Pool` | [ ] |
| F3 | `posttrade/pool.go` | `(p *Pool) Submit(ctx, job Job) error` | [ ] |
| F4 | `posttrade/pool.go` | `(p *Pool) Shutdown(ctx) (Stats, error)`: finish running, count queued | [ ] |
| F5 | `posttrade/retry.go` | `Backoff(attempt int, base, max time.Duration) time.Duration` with jitter | [ ] |
| F6 | `posttrade/retry.go` | `Retry(ctx, policy Policy, fn func(context.Context) error) error` | [ ] |
| F7 | `posttrade/dedupe.go` | `Seen.MarkIfNew(tradeID) bool`: idempotency, mutex-protected | [ ] |
| F8 | `posttrade/deadletter.go` | `DeadLetters.Add` / `List` | [ ] |
| F9 | `posttrade/fake.go` | Fake confirmation service: random failures + slowness | [ ] |
| F10 | `posttrade/*_test.go` | 30% failure rate → every trade done or dead-lettered, none twice | [ ] |

**Deliverable:** failure table in `docs/notes/M3.md` (failure → what happens → what we do).

---

## Module 4: Order gateway over TCP · 4 weeks · `docs/hld/M4.md`

**Goal:** real trader programs connect over TCP and send orders in your own protocol.

**Go you'll learn:** `net.Listen`, `net.Conn`, `bufio.Reader`/`Scanner`, `io.Reader`/`io.Writer`,
`io.ReadFull`, `encoding/binary`, deadlines, `net.Pipe` for tests, `strings`/`strconv` parsing.

**Systems you'll learn:** TCP is a byte stream, not messages (framing), partial reads, the 4 kinds
of timeout (connect, read, write, idle), heartbeats, slow clients, connection resets.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `gateway/text.go` | `ParseLine(line string) (engine.Command, error)` / `FormatEvent(ev) string` | [ ] |
| F2 | `gateway/server.go` | `NewServer(addr string, r *engine.Router) *Server` | [ ] |
| F3 | `gateway/server.go` | `(s *Server) Serve(ctx) error`: accept loop, stops on ctx | [ ] |
| F4 | `gateway/session.go` | `(s *session) readLoop(ctx)` with read deadline + heartbeat | [ ] |
| F5 | `gateway/session.go` | `(s *session) writeLoop(ctx)`: buffered outbox, drop slow clients | [ ] |
| F6 | `gateway/binary.go` | `Encode(w io.Writer, m Msg) error` / `Decode(r io.Reader) (Msg, error)` | [ ] |
| F7 | `cmd/trader/main.go` | Trader connects over TCP | [ ] |
| F8 | `gateway/*_test.go` | Tests with `net.Pipe`: half messages, bad input, silent client | [ ] |

**Deliverable:** `docs/protocol.md`: enough for someone to build a trader without your code.
**Done when:** 50 traders at once; killing one mid-message doesn't affect the others.

---

## Module 5: Market data feed · 3 weeks · `docs/hld/M5.md`

**Goal:** everyone watching gets every trade and price change, live and in order.

**Go you'll learn:** fan-out with channels, `select` with `default` (non-blocking send),
`sync.RWMutex`, `net.ListenMulticastUDP` / `net.UDPConn` (stretch).

**Systems you'll learn:** slow consumers (block vs drop vs buffer), snapshot + live updates,
sequence numbers and gap detection, TCP vs UDP.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `orderbook/book.go` | `(b *OrderBook) Depth(n int) (bids, asks []Level)` | [ ] |
| F2 | `marketdata/hub.go` | `NewHub(bufSize int) *Hub` | [ ] |
| F3 | `marketdata/hub.go` | `(h *Hub) Subscribe() (*Sub, Snapshot)` / `Unsubscribe` | [ ] |
| F4 | `marketdata/hub.go` | `(h *Hub) Publish(u Update)`: never blocks, drops slow subs | [ ] |
| F5 | `marketdata/client.go` | `(c *Client) Apply(u Update) error`: returns `ErrGap` | [ ] |
| F6 | `cmd/watcher/main.go` | Live top-of-book in the terminal | [ ] |
| F7 | `marketdata/udp.go` | Stretch: UDP multicast sender + receiver | [ ] |

**Deliverable:** `docs/adr/0002-slow-consumers.md`

---

## Module 6: Risk checks and rate limiting · 3 weeks · `docs/hld/M6.md`

**Goal:** block bad or abusive orders at the gateway, before matching.

**Go you'll learn:** `sync/atomic` (`atomic.Bool`, `atomic.Int64`), injecting a clock (small interface),
`testing/synctest`, benchmarks (`testing.B`), reading `golang.org/x/time/rate` source.

**Systems you'll learn:** token bucket vs leaky bucket vs windows, per-user state without a global
lock, kill switches, circuit limits.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `risk/bucket.go` | `NewTokenBucket(rate float64, burst int, now func() time.Time)` | [ ] |
| F2 | `risk/bucket.go` | `(b *TokenBucket) Allow() bool` | [ ] |
| F3 | `risk/checks.go` | `(c *Checker) Check(cmd engine.Command) error`: size, price band, open orders | [ ] |
| F4 | `risk/killswitch.go` | `BlockTrader(id)`, `HaltEntity(e)`, `IsBlocked(...)` with atomics | [ ] |
| F5 | `risk/*_test.go` | Fake clock tests; benchmark vs `x/time/rate` | [ ] |

**Deliverable:** `docs/notes/M6.md`: Knight Capital 2012 in half a page + what would've stopped it.

---

## Module 7: Observability · 3 weeks · `docs/hld/M7.md`

**Goal:** see what the exchange is doing while it runs.

**Go you'll learn:** `log/slog` (structured logs, handlers, `With`), `net/http`, `http.ServeMux`
route patterns, `net/http/httptest`, Prometheus `client_golang`, middleware as `func(http.Handler) http.Handler`.

**Systems you'll learn:** logs vs metrics vs traces, counters/gauges/histograms, percentiles,
the four golden signals, Docker Compose.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `cmd/exchange/main.go` | `slog` setup: JSON in prod, text in dev | [ ] |
| F2 | `metrics/metrics.go` | `New(reg prometheus.Registerer) *Metrics` | [ ] |
| F3 | `metrics/metrics.go` | `ObserveOrder`, `ObserveLatency(start time.Time)`, queue gauges | [ ] |
| F4 | `metrics/http.go` | `Handler() http.Handler`: `/metrics` + `/healthz` | [ ] |
| F5 | `docker-compose.yml` | Prometheus + Grafana + a dashboard | [ ] |

**Deliverable:** `docs/runbook.md`: "latency is high, what do I check first?"

---

## Module 8: Trade storage in Postgres · 4 weeks · `docs/hld/M8.md`

**Goal:** every trade saved reliably, even when the database is slow or down.

**Go you'll learn:** `pgx/v5` + `pgxpool` (or `database/sql`), `//go:embed` for SQL migrations,
context timeouts on every call, batch inserts, transactions, integration tests against a real DB.

**Systems you'll learn:** connection pools (and reading `database/sql`'s pool code), timeouts at each
layer, circuit breakers, MVCC, indexes and `EXPLAIN ANALYZE`, sizing a pool with numbers.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `store/migrations/*.sql` | `orders` and `trades` tables + indexes | [ ] |
| F2 | `store/store.go` | `Open(ctx, dsn string, cfg PoolConfig) (*Store, error)` | [ ] |
| F3 | `store/store.go` | `(s *Store) InsertTrades(ctx, []orderbook.Trade) error`: batch, dedupe by trade ID | [ ] |
| F4 | `store/store.go` | `TradeHistory(ctx, userID, limit)`, `DailyVolume(ctx, day)` | [ ] |
| F5 | `store/breaker.go` | `NewBreaker(...)`, `(b *Breaker) Do(ctx, fn) error`: closed/open/half-open | [ ] |
| F6 | `posttrade` | Post-trade pool writes through the breaker | [ ] |

**Deliverable:** `docs/notes/M8.md`: data model + pool sizing with measurements.
**Done when:** stopping Postgres mid-load loses no trades and matching never stops.

---

## Module 9: Journal and crash recovery · 5 weeks · `docs/hld/M9.md`

**Goal:** `kill -9` at any moment, restart, and get back exactly the same order books.

**Go you'll learn:** `os.File`, `bufio.Writer`, `(*os.File).Sync`, `hash/crc32`, `io.EOF` vs
`io.ErrUnexpectedEOF`, `encoding/binary`, `os/exec` (for the crash-test harness).

**Systems you'll learn:** write-ahead logging, what `fsync` really does, torn writes and checksums,
determinism (no `time.Now()` inside matching), snapshots + log replay.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `journal/record.go` | `encode(r Record) []byte` / `decode([]byte) (Record, error)`: length + CRC | [ ] |
| F2 | `journal/writer.go` | `Open(dir string, policy SyncPolicy) (*Writer, error)` | [ ] |
| F3 | `journal/writer.go` | `(w *Writer) Append(r Record) error`, `Sync() error`, `Rotate() error` | [ ] |
| F4 | `journal/replay.go` | `Replay(dir string, fn func(Record) error) error`: stops cleanly at a torn tail | [ ] |
| F5 | `orderbook/snapshot.go` | `(b *OrderBook) Snapshot() State` / `Restore(State) *OrderBook` | [ ] |
| F6 | `cmd/crashtest/main.go` | Start, `kill -9` at random, restart, compare books; 100 rounds | [ ] |

**Deliverable:** `docs/recovery.md` + `docs/postmortems/001.md` (a real bug you hit).

---

## Module 10: Performance · 4 weeks · `docs/hld/M10.md`

**Goal:** make it measurably faster, and prove every change with numbers.

**Go you'll learn:** `testing.B`, `b.ReportAllocs`, `benchstat`, `pprof` (CPU, heap),
`go tool trace`, `-gcflags=-m` (escape analysis), `sync.Pool`, `GOGC`/`GOMEMLIMIT`.

**Systems you'll learn:** benchmarking without fooling yourself, heap vs stack, GC pauses and p99,
CPU caches, the Go scheduler at a basic level.

| # | File | Function (sketch) | Status |
|---|---|---|---|
| F1 | `orderbook/bench_test.go` | `BenchmarkSubmit`, `BenchmarkMatchSweep`, `BenchmarkCancel` | [ ] |
| F2 | `engine/bench_test.go` | End-to-end orders/sec + p50/p99 | [ ] |
| F3 | `orderbook` | Replace sorted slices with a better structure, justified by numbers | [ ] |
| F4 | hot path | Cut allocations per order; before/after table | [ ] |

**Deliverable:** `docs/notes/M10-performance.md` (what, why, before/after numbers).
**Done when:** ≥3x faster on orders/sec or p99, backed by benchstat output.

---

## Module 11: Matching engine in Rust · 8 weeks · `docs/hld/M11.md`

Weeks 1–3: Rust Book ch. 1–10 + Rustlings. Then the order book in Rust (same test cases as M1),
one thread per stock, a `tokio` gateway speaking the M4 protocol, and the M10 benchmarks for Go vs Rust.

**Deliverable:** `docs/adr/000X-go-vs-rust.md` with the benchmark comparison.

---

## Module 12: Replication and failover · 6 weeks · `docs/hld/M12.md`

Weeks 1–4: MIT 6.5840 labs (MapReduce + Raft) as-is. Then: primary streams its journal (M9) to a
backup over TCP, the backup replays it live, takes over when the primary dies, and split brain is prevented.

**Deliverable:** `docs/notes/M12-raft.md` + `docs/adr/000X-failover.md`.
**Done when:** killing the primary under load loses no trade and does no trade twice.

---

## Module 13: Capstone · 2 weeks

`docker compose up` runs everything; polished README with the diagram; a 3-minute demo video;
a blog post with design, numbers and mistakes.

---

## Progress

| Module | Status | Started | Finished |
|---|---|---|---|
| M1 Order book | 🟨 | 2026-11-21 | |
| M2 Matching engine | ⬜ | | |
| M3 Post-trade pool | ⬜ | | |
| M4 Order gateway | ⬜ | | |
| M5 Market data feed | ⬜ | | |
| M6 Risk + rate limiting | ⬜ | | |
| M7 Observability | ⬜ | | |
| M8 Postgres storage | ⬜ | | |
| M9 Journal + recovery | ⬜ | | |
| M10 Performance | ⬜ | | |
| M11 Rust engine | ⬜ | | |
| M12 Replication | ⬜ | | |
| M13 Capstone | ⬜ | | |

⬜ not started · 🟨 in progress · ✅ done

## Resources (pick up when the module starts)

- **Go:** go.dev/doc/code, Effective Go, *100 Go Mistakes* (Harsanyi), *Concurrency in Go* (Cox-Buday),
  Go blog posts on pipelines, profiling, errors.
- **M1:** Zerodha Varsity Module 1. **M2:** Fowler, "The LMAX Architecture".
- **M3:** Marc Brooker, "Exponential Backoff and Jitter". **M4:** Beej's Guide, Cloudflare "Go net/http timeouts", NASDAQ OUCH.
- **M5:** NASDAQ ITCH. **M7:** Google SRE book, monitoring chapter. **M8:** *Use The Index, Luke*, *Release It!*.
- **M9:** DDIA ch. 3, Dan Luu "Files are hard". **M10:** Dave Cheney's High Performance Go.
- **M11:** Rust Book, Rustlings, *Rust Atomics and Locks*. **M12:** Raft paper, MIT 6.5840.
