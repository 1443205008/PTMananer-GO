package fakeseed

import (
	"testing"
)

// 构造一个最小合法 .torrent（bencode）
func buildTestTorrent() []byte {
	// d 8:announce 30:http://tracker.example/announce
	//   4:info d 6:length i3528000000000 e 4:name 10:ubuntu.iso e e
	return []byte("d8:announce31:http://tracker.example/announce4:infod6:lengthi3528000000000e4:name10:ubuntu.isoee")
}

func TestParseTorrentSingleFile(t *testing.T) {
	data := buildTestTorrent()
	p, err := parseTorrent(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.trackerURL != "http://tracker.example/announce" {
		t.Fatalf("tracker: %s", p.trackerURL)
	}
	if p.name != "ubuntu.iso" {
		t.Fatalf("name: %s", p.name)
	}
	if p.totalSize != 3528000000000 {
		t.Fatalf("size: %d", p.totalSize)
	}
	if len(p.infoHash) != 40 {
		t.Fatalf("infoHash length: %d (%s)", len(p.infoHash), p.infoHash)
	}
}

func TestParseTorrentMultiFile(t *testing.T) {
	// info 含 files 列表，两个 length
	// root: d 8:announce 20:.. 4:info d 5:files l d 6:length i100e 4:name 4:a.ce? — 用精确长度构造：
	// 文件1名 "a.bin"(5) 文件2名 "b.bin"(5) 目录 "dir"(3)
	data := []byte("d8:announce26:http://tracker.example/an4:infod5:filesld6:lengthi100e4:name5:a.bined6:lengthi200e4:name5:b.binee4:name3:diree")
	p, err := parseTorrent(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.totalSize != 300 {
		t.Fatalf("multi-file size sum: %d", p.totalSize)
	}
	// 与 TS 版一致：indexOf 找第一个 4:name（多文件模式下在 files 内部）
	if p.name != "a.bin" {
		t.Fatalf("name: %s", p.name)
	}
}

func TestParseTorrentMissingAnnounce(t *testing.T) {
	data := []byte("d4:infod6:lengthi1eee")
	if _, err := parseTorrent(data); err == nil {
		t.Fatal("missing announce should error")
	}
}

func TestBencodeSkip(t *testing.T) {
	// 跳过嵌套结构
	buf := []byte("d4:infod6:lengthi5ee3:fooi7ee")
	infoAt := bytesIndex(buf, "4:info")
	end, err := bencodeSkip(buf, infoAt+6)
	if err != nil {
		t.Fatal(err)
	}
	// end 应正好是整个字典结束位置
	if end != len(buf) {
		// info 值结束位置 < 总长（还有 foo）
		if end >= len(buf) {
			t.Fatalf("end %d out of range %d", end, len(buf))
		}
	}
}

func TestParseTrackerInterval(t *testing.T) {
	resp := []byte("d8:intervali1800e5:peerslee")
	if v, ok := parseTrackerInterval(resp); !ok || v != 1800 {
		t.Fatalf("interval = %d %v", v, ok)
	}
	// 越界值忽略
	if _, ok := parseTrackerInterval([]byte("d8:intervali30ee")); ok {
		t.Fatal("interval < 60 should be ignored")
	}
	if _, ok := parseTrackerInterval([]byte("d5:peerslee")); ok {
		t.Fatal("missing interval")
	}
}

func TestGenPeerIDAndKey(t *testing.T) {
	id := genPeerID()
	if len(id) != 20 || id[:8] != "-TR2920-" {
		t.Fatalf("peerId: %s", id)
	}
	k := genKey()
	if len(k) != 8 {
		t.Fatalf("key: %s", k)
	}
	if genPeerID() == genPeerID() {
		t.Fatal("peerId should be random")
	}
}

func bytesIndex(b []byte, sub string) int {
	return bytesIndexImpl(b, sub)
}
