package iris

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

// schema/golden.txt pins Go's bytes for every message; Rust (src/wire.rs) and
// TS (viewer) build the same values and must match byte for byte.
var update = flag.Bool("update", false, "rewrite schema/golden.txt")

type gold struct {
	name string
	kind uint8
	msg  []byte
}

func golds() []gold {
	hello := NewHello(HelloInput{W: 1280, H: 800, Dpr: 2, Credit: 2})
	return []gold{
		{"hello", 0, hello},
		{"credit", KCredit, NewCredit(CreditInput{Seq: 7, N: 1, DecodeUs: 1234})},
		{"pointer", KPointer, NewPointer(PointerInput{X: 100, Y: 200, Buttons: 1, Mods: 8})},
		{"wheel", KWheel, NewWheel(WheelInput{X: 10, Y: 20, Dx: -3, Dy: 120, Mods: 2})},
		{"key", KKey, NewKey(KeyInput{Sym: 0x61, Code: 30, Down: true})},
		{"commit", KCommit, NewCommit(CommitInput{Text: "héllo"})},
		{"clip", KClip, NewClip(ClipInput{Text: "copied"})},
		{"resize", KResize, NewResize(ResizeInput{W: 1024, H: 768, Dpr: 1.5})},
		{"refresh", KRefresh, NewRefresh(RefreshInput{Q: 90})},
		{"pick", KPick, NewPick(PickInput{Target: "T1"})},
		{"files", KFiles, NewFiles(FilesInput{Paths: [][]byte{[]byte("/home/sandbox/Downloads/a.txt"), []byte("/b")}})},
		{"cdp", KCdp, NewCdp(CdpInput{Id: 5, Json: `{"method":"Page.navigate"}`})},
		{"size", KSize, NewSize(SizeInput{W: 1280, H: 800, Dpr: 2, Mode: Browser})},
		{"tile", KTile, NewTile(TileInput{Seq: 9, W: 1280, H: 800, Q: 60, Last: true, Jpeg: []byte{0xff, 0xd8, 0xff, 0xd9}})},
		{"cursor", KCursor, NewCursor(CursorInput{Css: "pointer"})},
		{"point", KPoint, NewPoint(PointInput{X: 5, Y: 6})},
		{"want", KWant, NewWant(WantInput{Kind: 1, Many: true})},
		{"stats", KStats, NewStats(StatsInput{Fps: 30, Q: 60, Kbps: 1500, EncodeUs: 800, RttUs: 12000})},
		{"bye", KBye, NewBye(ByeInput{Code: 401, Reason: "ticket"})},
		{"open", Open, Hi(1, []byte("tkt"), hello)},
	}
}

func render() string {
	var b strings.Builder
	for _, g := range golds() {
		m := g.msg
		if g.name != "open" && g.kind != 0 {
			m = Tag(m, g.kind)
		}
		fmt.Fprintf(&b, "%s %d %s\n", g.name, g.kind, hex.EncodeToString(m))
	}
	return b.String()
}

func TestGolden(t *testing.T) {
	got := render()
	if *update {
		if err := os.WriteFile("schema/golden.txt", []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile("schema/golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("golden drift; run go test -run Golden -args -update\n%s", got)
	}
}

func TestReadBack(t *testing.T) {
	for _, g := range golds() {
		if g.kind == 0 || g.name == "open" {
			continue
		}
		if k := Kind(Tag(append([]byte(nil), g.msg...), g.kind)); k != g.kind {
			t.Errorf("%s: kind %d, want %d", g.name, k, g.kind)
		}
	}
	tl, _ := WrapTile(NewTile(TileInput{Seq: 9, W: 1280, H: 800, Q: 60, Last: true, Jpeg: []byte{1, 2}}))
	if tl.Seq() != 9 || tl.W() != 1280 || !tl.Last() || !bytes.Equal(tl.Jpeg(), []byte{1, 2}) {
		t.Fatal("tile round trip")
	}
	f, _ := WrapFiles(NewFiles(FilesInput{Paths: [][]byte{[]byte("a"), []byte("bc")}}))
	var ps []string
	f.Paths().EachBytes(func(_ int, b []byte) bool { ps = append(ps, string(b)); return true })
	if strings.Join(ps, ",") != "a,bc" {
		t.Fatalf("files %v", ps)
	}
}

func TestFrame(t *testing.T) {
	var w bytes.Buffer
	body := Body(1, Tag(NewRefresh(RefreshInput{Q: 90}), KRefresh))
	if err := Write(&w, Msg, body); err != nil {
		t.Fatal(err)
	}
	dir, got, err := Read(&w)
	if err != nil || dir != Msg || !bytes.Equal(got, body) {
		t.Fatalf("frame %d %x %v", dir, got, err)
	}
	id, m, ok := Split(got)
	if !ok || id != 1 || Kind(m) != KRefresh {
		t.Fatal("split")
	}
	if _, _, err := Read(bytes.NewReader([]byte{0, 0, 0, 0, 4})); err != ErrFrame {
		t.Fatalf("zero length accepted: %v", err)
	}
	if _, _, err := Read(bytes.NewReader([]byte{0xff, 0xff, 0xff, 0xff, 4})); err != ErrFrame {
		t.Fatalf("huge length accepted: %v", err)
	}
}
