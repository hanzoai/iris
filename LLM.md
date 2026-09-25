# iris

A sandbox screen over ONE ZAP stream. `iris` runs in the pod (`iris attach`
through the exec channel), the viewer runs in the app, and cloud's bridge
between them redeems the ticket and moves frames. No port, no page, no VNC.

| path | what |
|---|---|
| `schema/iris.zap` | the wire. `schema/golden.txt` pins Go's bytes; Rust and TS tests must match |
| `iris.go`, `iris_zap.go` | Go binding (zapgen) + framing; cloud imports `github.com/hanzoai/iris` |
| `src/` | the daemon (Rust): `cdp` source = chrome-headless-shell over a pipe |
| `viewer/` | `@hanzo/iris`: TS core (codec, credit, input) + `<Screen>` (web worker canvas; RN Skia) |
| `bench/` | probe page + harness for the design's benchmark table |

## Wire

Frame = `[u32 LE 1+len][dir][body]` (zap-proto/go transport). Over the
WebSocket the length is dropped: one binary message = `[dir][body]`.

- `dir 3` Open. body = rpc request envelope: Method 1 (`Iris.watch`),
  PromiseID = stream id, **Cap = ticket** (UTF-8), Payload = `Hello`.
- `dir 4` Msg. body = `[u32 LE stream][ZAP message]`; kind = header msgType
  (`flags >> 8`), numbers in the schema comments / `iris.go`.
- `dir 5` End. body = `[u32 LE stream]`.

Client speaks first. The bridge allows 5 s and 4 KiB before Open, redeems Cap
(single-use, 30 s, owner-minted), strips it, and passes the frame on with the
length prefix restored. iris never sees a ticket. Any failure is `Bye{Code}`
then End then close: 400 bad open, 401 ticket, 404/409 sandbox, 408 slow open,
413 big open, 502 iris gone, 503 no browser.

## Semantics

- **Size** first, then frames. Coordinates are CSS px in Size's space; Dpr ≤ 2.
- **Credit.** Hello.Credit is the window (viewer: 2, agent: 0). A frame is the
  tiles up to `Last`, and costs one credit. The viewer returns
  `Credit{Seq, N:1, DecodeUs}` per PRESENTED frame. At 0 nothing is sent and
  the next frame supersedes the last (the daemon holds one JPEG, never a
  queue); a viewer whose credit comes back after a skip gets the latest frame.
  Chrome's screencast ack is withheld while no viewer holds credit, so Chrome
  coalesces instead of encoding. Hidden viewer = no credit = no bytes.
- **Sharpen.** 200 ms after the last frame, one q90 `Page.captureScreenshot`
  replaces it (TurboVNC's lossless refresh, in JPEG).
- **Refresh{Q}** answers with one whole-screen Tile at Q, credit or not.
- **Resize** sets the viewport (last one wins); Size + a full frame follow.
- **Input.** Pointer is full state (DOM `buttons`, CDP modifier bits: Alt 1,
  Ctrl 2, Meta 4, Shift 8). Wheel dx/dy in CSS px, DOM sign. Key = X keysym +
  evdev code. Commit and client Clip insert text. A client's release on
  disconnect: its held buttons and keys are released.
- **Server Clip** = the page copied. **Cursor.Css** = the CSS cursor under the
  pointer. **Want{1,Many}** = a file chooser opened; answer `Files{Paths}`
  (paths in the sandbox). **Point** = another client's pointer.
- **Cdp{Id, Json}**: raw CDP `{"method","params"}` on the shown page (or the
  browser for `Target.*`/`Browser.*`); the reply is `Cdp{Id, response}`.
  Agents read `Accessibility.getFullAXTree` / `DOMSnapshot` here first and ask
  for pixels only with Refresh. A ticket holder already has exec, so raw CDP
  grants nothing new.

## Daemon

`iris attach` joins stdio to `/run/iris/sock` (dir 0700), starting
`iris serve` if nobody answers; serve signals readiness on a pipe, never a
poll. One Chrome per pod, started on first attach, `--remote-debugging-pipe`
(no port), `--no-sandbox` (gVisor; the pod is the boundary). Exits 5 min after
the last client. All state lives on one hub thread fed by a channel; its only
timers are the sharpen and idle deadlines.
