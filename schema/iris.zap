# iris: one screen, one stream. Open = dir 3 calling Iris.watch with the
# ticket in Cap and Hello as payload; every later message is dir 4, and its
# kind is the ZAP header msgType (flags >> 8). Offsets packed, little endian.
# Mods: Alt 1, Ctrl 2, Meta 4, Shift 8 (CDP order). Buttons: DOM order.

package iris

# Open payload. Credit = frames the viewer can hold; 0 = agent, no frames.
struct Hello
    W      u16
    H      u16
    Dpr    f32
    Credit u8

# client -> server

# 1: N more frames may be sent; Seq = last presented tile.
struct Credit
    Seq      u32
    N        u8
    DecodeUs u32

# 2: full pointer state, CSS px.
struct Pointer
    X       u16
    Y       u16
    Buttons u8
    Mods    u8

# 3
struct Wheel
    X    u16
    Y    u16
    Dx   i16
    Dy   i16
    Mods u8

# 4: X keysym + evdev code.
struct Key
    Sym  u32
    Code u16
    Down bool
    Mods u8

# 5: IME commit / paste.
struct Commit
    Text text

# 7
struct Resize
    W   u16
    H   u16
    Dpr f32

# 8: one whole-screen Tile at quality Q, credit or not.
struct Refresh
    Q u8

# 9: show this target (tab).
struct Pick
    Target text

# 10: answer to Want; paths inside the sandbox.
struct Files
    Paths list<text>

# both ways

# 6: server->client = page copied; client->server = paste.
struct Clip
    Text text

# 11: raw CDP. Client Id is echoed on the reply.
struct Cdp
    Id   u32
    Json text

# server -> client

# 32: Mode 1 browser, 2 desktop.
struct Size
    W    u16
    H    u16
    Dpr  f32
    Mode u8

# 33: the only frame type; a keyframe is tiles covering the screen.
struct Tile
    Seq  u32
    X    u16
    Y    u16
    W    u16
    H    u16
    Q    u8
    Last bool
    Jpeg bytes

# 34
struct Cursor
    Css  text
    W    u16
    H    u16
    Hx   u16
    Hy   u16
    Rgba bytes

# 35: another client's pointer.
struct Point
    X u16
    Y u16

# 36: Kind 1 = file chooser.
struct Want
    Kind u8
    Many bool

# 37
struct Stats
    Fps      u8
    Q        u8
    Kbps     u32
    EncodeUs u32
    RttUs    u32

# 38
struct Bye
    Code   u16
    Reason text

interface Iris
    watch(req: Hello)
