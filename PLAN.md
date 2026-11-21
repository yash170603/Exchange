# Exchange: the 1-year plan

**What:** a small stock exchange. Traders connect, send buy and sell orders, the exchange
matches them fairly, broadcasts prices, saves trades, and survives crashes.

**Why:** every piece of an exchange forces you to learn one real systems skill: concurrency,
networking, timeouts, rate limiting, databases, crash recovery, performance, Rust, replication.

**Budget:** ~1 hour a day, ~5 hours a week, 52 weeks. Week counts below are estimates.
If a module takes longer, that's fine. **Keep the order, not the dates.**

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
                        │ Post-trade pool │ M3 (your job runner)
                        └────────┬────────┘
                                 ▼
                           Postgres  M8

  Across everything: M7 logs, metrics, dashboards · M10 performance
  Later: M11 engine rewritten in Rust · M12 backup exchange that takes over on crash
```

## Target folder layout

```text
exchange/
├── CLAUDE.md, PLAN.md, README.md
├── go.mod
├── Makefile
├── cmd/
│   ├── exchange/      ← main program: wires all modules together
│   └── trader/        ← fake trader that sends orders (load generator)
├── orderbook/         ← M1
├── engine/            ← M2
├── posttrade/         ← M3
├── gateway/           ← M4
├── marketdata/        ← M5
├── risk/              ← M6
├── metrics/           ← M7
├── store/             ← M8
├── journal/           ← M9
├── docs/
│   ├── adr/           ← decision records (one file per big decision)
│   └── notes/         ← your notes for each module
└── rust/              ← M11
```

Don't create these folders up front. Create each one when its module starts.

---

## How every module runs

1. **Day 1: design first.** Write a half-page design in `docs/notes/MX.md`: what it does,
   boxes and arrows, types, how it can fail. Ask Claude to review the design *before* coding.
2. **Build in small runnable steps.** It should compile and the tests should pass at the end
   of every session. Commit after each step.
3. **Break it on purpose.** Slow it down, kill it, flood it. Watch what happens.
4. **Last day: write it up.** Finish the module's written deliverable, then explain the module
   out loud in 3 sentences as if to a non-engineer. Ask Claude to quiz you.
5. **Take it to work.** Each module has a "slice tie-in": look for the same thing in real code.

**AI rule:** during the hour, Claude explains, hints and reviews. It doesn't write the
module's code. (See `CLAUDE.md`.)

### Words used in this plan

- **ADR (Architecture Decision Record):** a short file explaining one decision: the problem,
  the options, what you picked and why. Senior engineers write these so future readers
  (including future you) know *why* the code looks the way it does.
- **Runbook:** step-by-step instructions for what to do when something breaks.
- **Postmortem:** a write-up after something broke: what happened, why, what changes now.
  It focuses on causes, not on blaming people.
- **p99 latency:** sort all request times; p99 is the one that 99% are faster than. It shows
  how bad the worst cases are, which averages hide.

---

## Module 0: Project setup · 1 week

**Goal:** a clean repo you can build, test and run with one command.

**Build**
- [ ] Rename folder + module path, delete the old scaffolding (`types/`, `channels/`, `workers/`, `JobProcessor/`). Keep your `Job`/`Result` ideas for M3
- [ ] `cmd/exchange/main.go` that prints "exchange starting"
- [ ] `Makefile` with `test`, `vet`, `run` targets
- [ ] GitHub repo + GitHub Actions running `go vet` and `go test -race ./...` on every push
- [ ] `README.md`: one paragraph on what this is, and how to run it

**Technical things to learn**
- Go modules: `go.mod`, module path, how imports work, `go mod tidy`
- Packages: what `package main` means, exported vs unexported (capital letters)
- Tooling: `go build`, `go run`, `go test`, `go vet`, `gofmt`
- What CI is and why every push should run tests

**Non-technical things to learn**
- Git habits: small commits, messages that say *why*, branches even when working alone
- Writing a README someone else can follow

**Done when:** a fresh clone runs `make test` green, and CI shows a green tick.

**Resources:** go.dev/doc/code ("How to Write Go Code"), Effective Go.

---

## Module 1: Order book · 3 weeks

**Goal:** given orders one at a time, match them fairly and produce trades. Pure logic:
no goroutines, no network.

**Build**
- [ ] `Order`, `Trade`, `Side` types; `OrderBook` with `Submit(order) []Trade`
- [ ] Price-time priority: best price first, then oldest first
- [ ] Partial fills, and one order sweeping multiple price levels
- [ ] `Cancel(orderID) bool`
- [ ] Table-driven tests for every case (no match, full, partial, sweep, same-price ordering, cancel)
- [ ] Market orders (buy/sell at whatever price is available)

**Technical things to learn**
- Structs, methods, pointer vs value receivers (and why it matters here)
- Slices for real: `append`, capacity, why removing from the middle is costly
- Maps; why money is `int64` paise, never float (`0.1 + 0.2 != 0.3`)
- Table-driven tests, `t.Run` subtests, test helpers

**Non-technical things to learn**
- **Domain:** what an exchange does, bid/ask, spread, limit vs market orders, price-time priority
- Turning fuzzy rules into exact test cases *before* writing code

**Written deliverable:** `docs/notes/M1.md`, the matching rules in your own words, with one worked example.

**Done when:** every listed test passes, and you can trace a sweep example by hand.

**Slice tie-in:** find how money amounts are stored in slice's code. int? decimal? float?

**Resources:** Zerodha Varsity (Module 1), *Trading and Exchanges* by Larry Harris (chapters on order books).

---

## Module 2: Matching engine · 3 weeks

**Goal:** many stocks matched at the same time, each stock handled by exactly one goroutine.

**Build**
- [ ] `Engine`: one goroutine per stock, owns one `OrderBook`, reads orders from a channel
- [ ] `Router`: sends each incoming order to the right stock's channel
- [ ] Every order gets a sequence number on arrival (proves arrival order)
- [ ] Results (trades, acks, rejects) come out on an output channel
- [ ] `cmd/trader`: fake trader firing N random orders across stocks
- [ ] Clean shutdown with `context`: stop accepting, drain what's queued, exit
- [ ] Print orders/sec at the end of a run

**Technical things to learn**
- Goroutines, unbuffered vs buffered channels, `select`, closing channels (who closes what)
- `context.Context`: cancellation, how shutdown spreads through the program
- Why one goroutine per stock beats a worker pool + lock (fairness, no lock waiting, same result every time)
- **Backpressure:** what happens when a channel is full, and whether the sender should wait or reject
- `go test -race`, goroutine leaks and how to spot them

**Non-technical things to learn**
- Write your **first ADR**: "One goroutine per stock instead of a worker pool"
- Explaining a design choice by comparing the options, not just stating the answer

**Written deliverable:** `docs/adr/0001-one-goroutine-per-stock.md`

**Done when:** 100k random orders across 10 stocks run with `-race` clean, and Ctrl+C exits without losing queued orders.

**Slice tie-in:** find a Kafka consumer at slice. Does message order matter there? How is it kept?

**Resources:** Martin Fowler, "The LMAX Architecture"; *Concurrency in Go* (Cox-Buday), chapters 1–4.

---

## Module 3: Post-trade processor · 3 weeks

**Goal:** after a trade, do the slow side work (confirmations, saving) with a worker pool,
without slowing down matching and without doing anything twice. Your original job runner lives here.

**Build**
- [ ] Worker pool: N workers read trade events from a channel
- [ ] Each job has a timeout (`context.WithTimeout`)
- [ ] Retries with exponential backoff + jitter (random extra wait)
- [ ] Jobs that keep failing go to a "dead letter" list instead of retrying forever
- [ ] **Idempotency:** running the same trade event twice has the same effect as once
- [ ] Graceful shutdown: running jobs finish, queued ones are counted, nothing is silently dropped
- [ ] Fake slow/failing "confirmation service" to test against

**Technical things to learn**
- `sync.WaitGroup`, `sync.Mutex`, `errgroup`
- Timeouts: a timeout only works if the slow code checks `ctx.Done()`
- Why retries need backoff and jitter (otherwise all retries hit at the same moment)
- At-least-once vs exactly-once processing, and why idempotency is the practical answer
- Dead letter queues

**Non-technical things to learn**
- **Failure-mode thinking:** for each step, ask "what if this is slow? down? runs twice?"
- Write it down as a table: failure → what happens → what we do

**Written deliverable:** failure-mode table in `docs/notes/M3.md`

**Done when:** with the fake service failing 30% of the time, every trade ends up either done or in the dead letter list, with no duplicates.

**Slice tie-in:** how does slice avoid charging a customer twice when a request is retried? (Idempotency keys. You may have seen these at Razorpay.)

**Resources:** Go blog "Go Concurrency Patterns: Pipelines and cancellation"; AWS Architecture blog "Exponential Backoff and Jitter" (Marc Brooker); *100 Go Mistakes*, concurrency chapters.

---

## Module 4: Order gateway (TCP) · 4 weeks

**Goal:** real trader programs connect over TCP and send orders using a protocol you design.

**Build**
- [ ] TCP server: one goroutine per connected trader
- [ ] Your own message protocol: start text-based (one order per line), then a binary version
- [ ] Message framing: knowing where one message ends and the next starts
- [ ] Validate every order (positive qty, known stock, sane price); reject with a reason
- [ ] Read/write deadlines; disconnect traders that go silent (heartbeats)
- [ ] Handle a slow trader that doesn't read its responses (don't let it block the engine)
- [ ] `cmd/trader` now connects over TCP

**Technical things to learn**
- TCP basics: connections, the 3-way handshake, why TCP is a byte stream, not messages
- `net.Listen`, `net.Conn`, `bufio`, `encoding/binary`, endianness
- The different timeouts: connect, read, write, idle. What happens without each one
- Partial reads and writes; connection resets; `TIME_WAIT`
- Testing network code with `net.Pipe`

**Non-technical things to learn**
- **Write a protocol spec** (`docs/protocol.md`) that someone could build a trader from without reading your code
- Reading real specs: skim NASDAQ's OUCH spec to see how a real exchange documents this

**Written deliverable:** `docs/protocol.md`

**Done when:** 50 trader programs connect at once and trade; killing one mid-message doesn't affect the others.

**Slice tie-in:** pick one outgoing HTTP call in a slice service. What are its timeouts? What's the connection pool size?

**Resources:** *Beej's Guide to Network Programming* (C, but best explanation of sockets); Cloudflare blog "The complete guide to Go net/http timeouts"; NASDAQ OUCH spec.

---

## Module 5: Market data feed · 3 weeks

**Goal:** anyone watching gets every trade and price change, live, in the correct order.

**Build**
- [ ] Broadcast trades and order book changes to all subscribers
- [ ] Sequence numbers on every message, so watchers can detect a missed message (a "gap")
- [ ] Snapshot on join (full book) + live updates after
- [ ] Slow watcher policy: drop them, never let them slow everyone else
- [ ] `cmd/watcher`: prints a live top-of-book view
- [ ] Stretch: UDP multicast version (what real exchanges use)

**Technical things to learn**
- Fan-out (one sender, many receivers) with goroutines and channels
- Slow consumers: block vs drop vs buffer, and the cost of each
- Snapshot + incremental updates, gap detection and recovery
- TCP vs UDP: when losing a packet is fine, and how you recover when it isn't

**Non-technical things to learn**
- **Domain:** what market data is, why it must be fair (everyone sees it at the same time), level 1 vs level 2 data
- ADR: "What we do with slow watchers"

**Written deliverable:** `docs/adr/0002-slow-consumers.md`

**Done when:** a deliberately slow watcher gets disconnected while 20 others keep up, and a watcher that misses messages detects the gap.

**Resources:** NASDAQ ITCH spec (skim the message types); Zerodha Varsity on market depth.

---

## Module 6: Pre-trade risk and rate limiting · 3 weeks

**Goal:** stop bad or abusive orders at the gateway before they reach matching.

**Build**
- [ ] Token bucket rate limiter per trader (built by you, then compared with `golang.org/x/time/rate`)
- [ ] Price bands: reject orders too far from the last price (like NSE circuit limits)
- [ ] Max order size and max open orders per trader
- [ ] Kill switch: instantly block one trader, or halt one stock
- [ ] Clear reject reasons sent back to the trader

**Technical things to learn**
- Rate limiting algorithms: token bucket, leaky bucket, fixed vs sliding window
- Doing these checks fast: no locks shared across traders where avoidable
- Testing time-based code without `time.Sleep` (injecting a fake clock)

**Non-technical things to learn**
- **Domain:** circuit limits, trading halts, fat-finger errors (and real incidents caused by them)
- Read one real exchange incident write-up (e.g. Knight Capital 2012) and write a half-page summary

**Written deliverable:** `docs/notes/M6.md` with the incident summary + what in your design would have stopped it

**Done when:** a trader flooding 10k orders/sec gets limited, while others are unaffected; a halted stock rejects all orders.

**Slice tie-in:** where does slice rate-limit (API gateway? per user? per partner?) and what happens when the limit is hit?

**Resources:** `x/time/rate` source code; Zerodha Varsity on circuit limits; SEC report on Knight Capital.

---

## Module 7: Observability · 3 weeks

**Goal:** see what the exchange is doing while it runs, and find problems from the outside.

**Build**
- [ ] Structured logs with `log/slog` (with order IDs, stock, trader)
- [ ] Prometheus metrics: orders/sec, trades/sec, rejects by reason, queue lengths, connected traders
- [ ] Latency histogram: time from order received → result sent
- [ ] Grafana dashboard (Prometheus + Grafana via Docker Compose)
- [ ] Health check endpoint

**Technical things to learn**
- Logs vs metrics vs traces, and when each one helps
- Counters, gauges, histograms; why averages lie and percentiles don't
- Docker and Docker Compose basics
- The "four golden signals": latency, traffic, errors, saturation (how full things are)

**Non-technical things to learn**
- Write a **runbook**: "Latency is high, what do I check first?"
- Deciding what to alert on (and what not to: alerts nobody acts on get ignored)

**Written deliverable:** `docs/runbook.md`

**Done when:** you can run a load test, open Grafana, and point at the exact queue that's backing up.

**Slice tie-in:** open slice's dashboards for your service. Which golden signals are there? Which are missing?

**Resources:** Google SRE book, chapter "Monitoring Distributed Systems"; Prometheus docs on histograms.

---

## Module 8: Trade storage in Postgres · 4 weeks

**Goal:** every trade gets saved to Postgres reliably, even when the database is slow or down.

**Build**
- [ ] Schema for orders and trades; migrations
- [ ] Post-trade pool (M3) writes trades to Postgres
- [ ] Tune the connection pool: max open, max idle, max lifetime, with reasons for each number
- [ ] Query timeouts on every DB call
- [ ] Batch inserts; measure the speedup vs one-at-a-time
- [ ] Circuit breaker (built by you): when Postgres is failing, stop hammering it, buffer, retry later
- [ ] Queries: a trader's trade history, daily volume per stock; make them fast with indexes

**Technical things to learn**
- **Connection pooling, in depth:** what a connection costs, what happens when the pool is empty,
  read `database/sql` source (`sql.go`) to see how Go's pool really works
- Timeouts at each layer: pool wait, query, network
- Circuit breaker states: closed → open → half-open, and why each exists
- Transactions, isolation levels, MVCC (how Postgres lets readers and writers not block each other)
- Indexes, `EXPLAIN ANALYZE`, why an index can be ignored
- Postgres connection limits and why tools like PgBouncer exist

**Non-technical things to learn**
- **Data modeling doc:** tables, why each column exists, which queries each index serves
- Making a decision with numbers: "pool size 20 because I measured X", not "20 felt right"

**Written deliverable:** `docs/notes/M8.md` with the data model + a pool sizing write-up with measurements

**Done when:** stopping Postgres mid-load loses no trades (they're saved once it's back), and the exchange keeps matching the whole time.

**Slice tie-in:** check your service's DB pool settings. Can you explain each number? Look for a slow query and run `EXPLAIN ANALYZE` on it (on a non-production DB).

**Resources:** *Use The Index, Luke* (free online); *Release It!* (Nygard), stability patterns chapter; Postgres docs on MVCC; `database/sql` source.

---

## Module 9: Journal and crash recovery · 5 weeks

**Goal:** kill the exchange with `kill -9` at any moment, restart it, and get back exactly the same order books.

**Build**
- [ ] Journal: every order is written to an append-only file *before* the engine processes it
- [ ] Binary record format with length + checksum (to detect half-written records)
- [ ] `fsync` strategies: every write vs batched; measure the latency cost of each
- [ ] Recovery: on start, replay the journal to rebuild every order book
- [ ] Snapshots: save full book state periodically so replay doesn't start from zero
- [ ] Journal file rotation and cleanup of old files after a snapshot
- [ ] Crash test script: random `kill -9` in a loop, check books match every time

**Technical things to learn**
- Write-ahead logging (WAL): why databases write the log first
- What `fsync` really does, the page cache, why "written" doesn't mean "on disk"
- Torn writes, checksums (CRC32), detecting a corrupt tail
- Why the engine must behave the same way every time for replay to work
  (no `time.Now()` or random numbers inside matching)
- Snapshots + log = how Postgres, Kafka and Redis all recover

**Non-technical things to learn**
- Write a **disaster recovery runbook**: "The exchange crashed. Here's how to restart and verify it."
- Write a **postmortem** for one bug you hit during this module, as if it happened in production

**Written deliverable:** `docs/recovery.md` + `docs/postmortems/001.md`

**Done when:** 100 random `kill -9` runs, and recovered state matches every time.

**Resources:** *Designing Data-Intensive Applications* (Kleppmann), chapter 3; *Database Internals* (Petrov); Dan Luu, "Files are hard"; CMU 15-445 lectures on logging and recovery (free on YouTube).

---

## Module 10: Performance · 4 weeks

**Goal:** make the exchange measurably faster, and prove every improvement with numbers.

**Build**
- [ ] Benchmarks (`testing.B`) for the order book and engine
- [ ] Baseline numbers: orders/sec, p50/p99 latency, allocations per order
- [ ] CPU and memory profiles with `pprof`; find the top 3 hot spots
- [ ] Replace the sorted slices in the order book with a better structure (justify it with numbers)
- [ ] Cut allocations in the hot path (reuse objects, `sync.Pool` where it helps)
- [ ] Look at GC pauses and their effect on p99
- [ ] Before/after table for every change

**Technical things to learn**
- Benchmarking properly: `benchstat`, warmup, avoiding fooling yourself
- `pprof` (CPU, heap, allocs), `go tool trace`
- Escape analysis: when Go puts things on the heap and why it costs
- How the garbage collector works and why it causes latency spikes
- CPU caches: why a slice can beat a "faster" tree in practice
- The Go scheduler (G, M, P) at a basic level

**Non-technical things to learn**
- **Performance write-up:** what you measured, what you changed, the numbers. One page.
- Knowing when to stop: "fast enough" is a decision, not a feeling

**Written deliverable:** `docs/notes/M10-performance.md`

**Done when:** at least 3x faster than baseline on orders/sec or p99, backed by benchmark output.

**Resources:** Go blog "Profiling Go Programs"; Dave Cheney, "High Performance Go Workshop"; *100 Go Mistakes*, performance chapters.

---

## Module 11: Matching engine in Rust · 8 weeks

**Goal:** rebuild the order book and engine in Rust, plug it into the same system, and compare it with Go.

**Build**
- [ ] Weeks 1–3: learn Rust basics with small exercises (Rust Book chapters 1–10, Rustlings)
- [ ] Order book in Rust, passing the same test cases as M1
- [ ] Engine with one thread per stock (`std::thread` + channels), then a `tokio` version of the gateway
- [ ] Speaks the same protocol as M4, so the Go traders work with it unchanged
- [ ] Same benchmarks as M10; compare Go vs Rust (throughput, p99, memory)

**Technical things to learn**
- Ownership, borrowing, lifetimes: what the compiler is protecting you from
- `Result`, `Option`, error handling with `?`
- Traits, generics, enums with data (great for order types)
- Threads, channels, `Arc`/`Mutex`; why Rust can catch data races at compile time
- Async Rust with `tokio`
- `cargo`, `cargo test`, `cargo bench` (criterion)

**Non-technical things to learn**
- Learning a new language fast: build the same thing you already know
- ADR: "When would we choose Rust over Go for this system?"

**Written deliverable:** `docs/adr/000X-go-vs-rust.md` with the benchmark comparison

**Done when:** the Rust engine passes all M1 tests and trades with the Go traders over TCP.

**Resources:** *The Rust Programming Language* (the Rust Book); Rustlings; Tokio tutorial; *Rust Atomics and Locks* (Mara Bos); later *Rust for Rustaceans* (Gjengset).

---

## Module 12: Replication and failover · 6 weeks

**Goal:** run a backup exchange that copies everything and takes over if the main one dies.

**Build**
- [ ] Weeks 1–4: MIT 6.5840 labs on their own, MapReduce + Raft (leader election, log replication).
      The labs' test suite is the point, so do them as-is
- [ ] Primary sends its journal (M9) to a backup over TCP; backup replays it live
- [ ] Failover: primary dies → backup takes over → traders reconnect
- [ ] Make sure two machines never both act as the primary ("split brain")
- [ ] Stretch: use your Raft from the labs to choose the primary

**Technical things to learn**
- Replication: primary/backup, synchronous vs asynchronous, what you can lose in each
- Consensus and Raft: leader election, terms, log matching
- Split brain, and why "just ping it" isn't enough to know a machine is dead
- Network partitions and the CAP trade-off, in plain terms

**Non-technical things to learn**
- Explaining a hard topic simply: write "Raft in 1 page" for a junior engineer

**Written deliverable:** `docs/notes/M12-raft.md` + `docs/adr/000X-failover.md`

**Done when:** killing the primary under load → backup takes over, no trade lost, no trade done twice.

**Resources:** MIT 6.5840 course site and labs; Raft paper ("In Search of an Understandable Consensus Algorithm"); thesecretlivesofdata.com/raft.

---

## Module 13: Capstone · 2 weeks

**Goal:** turn a year of work into something people can see.

**Build**
- [ ] `docker compose up` starts the whole system: exchange, traders, watcher, Postgres, Grafana
- [ ] Polished README with the architecture diagram and how to run it
- [ ] A 3-minute demo video: load test, Grafana, kill the primary, failover

**Non-technical things to learn**
- **Blog post:** "I built a stock exchange to learn systems programming": design, numbers, mistakes
- Turning the project into resume bullets with real numbers
- Presenting it: practise explaining it in 2 minutes and in 20 minutes

**Done when:** a stranger can clone the repo and see it working in 5 minutes.

---

## Progress

| Module | Status | Started | Finished |
|---|---|---|---|
| M0 Setup | ⬜ | | |
| M1 Order book | ⬜ | | |
| M2 Matching engine | ⬜ | | |
| M3 Post-trade processor | ⬜ | | |
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
