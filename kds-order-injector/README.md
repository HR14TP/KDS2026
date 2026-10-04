# kds-order-injector

A standalone CLI that injects test orders straight into a running
`kdswebserver` instance, the same way the POS ("Sally") integration does when
it POSTs to `/order`. Useful for testing a KDS terminal (order flow, screens,
production timings, etc.) without needing an actual POS terminal or print
feed running.

It's a single static Go binary with no runtime dependencies - copy it to any
Debian/Ubuntu machine (or the KDS terminal itself) and run it.

## Build

On the target machine (or any Linux/amd64 machine, since the binary is
statically linked and has no CGO dependency):

```bash
cd kds-order-injector
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kds-order-injector-linux .
```

For an arm64 device (e.g. a tablet), set `GOARCH=arm64` instead.

## GUI

For pasting orders in interactively instead of using flags, launch the
browser-based GUI:

```bash
./kds-order-injector-linux -gui
```

This starts a local server (default `127.0.0.1:8787`) and opens it in your
default browser. It has three tabs:

- **Paste / File** - paste a single order object or an array of orders as
  JSON (or drag-and-drop / browse for a `.json` file, which loads its
  contents into the box for editing before sending), then **Send**.
- **Raw Ticket Text** - paste the raw text a POS ticket would print (table,
  staff, items, etc.) and it's parsed the same way kds-poller parses a real
  print feed, then converted into an order. Use **Preview parsed order** to
  see the resulting JSON before sending. See "Raw ticket text format" below.
- **Random Test Orders** - generate and send N random orders with an
  optional fixed table/waiter and an interval between sends, same as the
  `-count`/`-interval`/`-table`/`-waiter`/`-items`/`-seed` CLI flags.

The host field at the top applies to both tabs, is remembered between runs,
and has its own **Ping** button. Results (order id, table, waiter, item
count, success/failure) show up in the panel on the right as they're sent.

Use `-listen` to change the address/port it binds to, e.g. `-listen
0.0.0.0:8787` to reach it from another machine on the network.

## CLI usage

Point `-host` at the machine running `kdswebserver` (default port `2020`).
Running it against `localhost` on the KDS terminal itself works too.

Check connectivity first:

```bash
./kds-order-injector-linux -host http://localhost:2020 -ping
```

Send 5 randomly generated test orders, 2 seconds apart:

```bash
./kds-order-injector-linux -host http://localhost:2020 -count 5 -interval 2s
```

Send a specific order (or list of orders) from a JSON file - see
`sample-order.json` for the shape:

```bash
./kds-order-injector-linux -host http://localhost:2020 -file sample-order.json
```

Send a raw ticket/print-feed text file (see "Raw ticket text format" below):

```bash
./kds-order-injector-linux -host http://localhost:2020 -ticket-file ticket.txt
```

Send that same raw ticket text to the target terminal's local NATS instead,
so its own `kds-poller` parses it exactly like a real print feed (see "Raw
text via NATS" below):

```bash
./kds-order-injector-linux -ticket-file ticket.txt -nats -nats-addr 127.0.0.1:4222
```

### Flags

| Flag        | Default                 | Description |
|-------------|--------------------------|-------------|
| `-host`     | `http://localhost:2020` | Base URL of the kds-webserver to inject into |
| `-count`    | `1`                      | Number of orders to send |
| `-interval` | `500ms`                  | Delay between orders when sending more than one |
| `-file`     | (none)                   | JSON file with one order object or an array of orders, sent verbatim instead of generating random ones |
| `-ticket-file` | (none)                | Raw ticket/print-feed text file - parsed and sent as one order (takes priority over `-file`/random generation) |
| `-screen`   | `Kitchen`                | Fallback product center/screen for a `-ticket-file` order when the ticket has no screen marker of its own (ignored when `-nats` is set) |
| `-nats`     | off                      | With `-ticket-file`, publish the file's raw bytes to NATS subject `incoming` on `-nats-addr` instead of parsing locally and POSTing JSON to `/order` |
| `-nats-addr`| `127.0.0.1:4222`        | Target terminal's local NATS address (used with `-nats`) |
| `-table`    | random                   | Table name for generated orders |
| `-waiter`   | random                   | Waiter name for generated orders |
| `-items`    | random 1-4               | Number of line items per generated order |
| `-seed`     | time-based               | RNG seed for generated orders, for reproducible runs |
| `-ping`     | off                      | Just check connectivity against `<host>/ping` and exit |
| `-verbose`  | off                      | Print each request/response body |
| `-gui`      | off                      | Launch the browser-based GUI instead (ignores every other flag except `-listen`) |
| `-listen`   | `127.0.0.1:8787`         | Address the GUI server binds to (used with `-gui`) |

Each injected order gets a unique `id` (`TESTORDER-<timestamp>` for generated
orders), since `kdswebserver` matches incoming orders by `id` and treats a
repeat as an update to the existing ticket rather than a new one.

## Raw text via NATS (`-nats`)

By default, `-ticket-file` parses the ticket text *in this tool* (a ported
copy of `kds-poller`'s regexes/heuristics - see "Raw ticket text format"
below) and POSTs the resulting JSON straight to `kdswebserver`'s `/order`.
That's convenient, but it skips `kds-poller` entirely, so none of its
existing-order handling runs: merging into an already-open ticket, table
transfers, or voiding a previously-created item. The ported copy can also
drift from the original over time.

With `-nats`, the tool instead connects to the target terminal's own local
NATS (`-nats-addr`, default `127.0.0.1:4222` - matches `kds-poller`'s
`natsLocalPort`) and publishes the file's raw, unparsed bytes to subject
`incoming` - the exact subject and payload shape a real print-feed capture
publishes (`kds-poller-main/pkg/nats_coms.go`'s `NC.Subscribe("incoming",
...)`). That terminal's actual, unmodified `kds-poller` parses it from
there via `ProcessMessage` in `engine.go`, so this exercises the real
parser and its full order-handling logic rather than this tool's copy -
most useful for testing a ticket that merges into or voids against an
order already open on that terminal.

In a multi-terminal rig each terminal may run its own local NATS on a
different port, so always point `-nats-addr` at the specific terminal
whose `kds-poller` should own the parse.

## Raw ticket text format

`kds-poller` doesn't take JSON at all - it watches the real POS print feed
and parses the raw ticket text line by line (regexes and heuristics in
`kds-poller-main/pkg/process.go` and `engine.go`) directly into the SQLite
DB. `ticket_text.go` in this tool ports that same set of regexes/heuristics
so a pasted ticket parses the same way when `-nats` is *not* used:

- `  * profile *` - order profile (e.g. "dine in"), on its own line
- `* * screen *  *` - product center/screen for the whole ticket
- `table N` / `order N` / `tafel N` / `tisch N` - table name
- `ipad<n>/<name>` - waiter/staff name
- `HH:MM ... <ref>` - the order's reference/ticket id
- `N x item name` - a line item; a following `-=-` line closes it
- `consumer:` / `delivery time:` - consumer name / delivery time

Table/staff/profile/screen/reference each latch onto the *first* line that
matches, same as the real parser. See `sample-ticket.txt` for a working
example - try **Preview parsed order** in the GUI on it.

**Fidelity note:** the item/sub-item split (is this line a new dish, or a
modifier of the one above it?) is decided in the original by exact byte
positions in the raw bytes a real thermal printer feed produces - including
control bytes a browser textarea generally can't hold. Pasting hand-typed
ticket text through this tool, every `N x ...` line comes through as its own
flat top-level item rather than nested under the item before it. Table,
waiter, reference, profile, screen, and the item list itself (names and
quantities) still parse correctly either way - only the main-item/sub-item
nesting is affected, and only for hand-typed input.

## Caution

This talks directly to a live `kdswebserver`'s production order-ingestion
endpoint - anything sent lands on the KDS board like a real order. Don't
point `-host` at a terminal that's actively running a live service unless
that's actually the intent (e.g. deliberately testing on it), since injected
tickets will show up alongside real ones.
