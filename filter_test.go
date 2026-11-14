package packetepoch

import "testing"

func TestFirstPacket(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 10}) {
		t.Fatal("first packet rejected")
	}
}

func TestTwoSpeakersFirstPacketAtMaxSeq(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 65535}) {
		t.Fatal("first packet from speaker a rejected")
	}
	if !f.Accept(Packet{Speaker: "b", Epoch: 1, Seq: 65535}) {
		t.Fatal("first packet from speaker b rejected")
	}
}

func TestSequentialWrap(t *testing.T) {
	f := &Filter{}
	f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 65534})
	for i := 0; i < 4; i++ {
		seq := uint16(65535 + i) // 65535, 0, 1, 2
		if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: seq}) {
			t.Fatalf("seq %d rejected across wrap", seq)
		}
	}
}

func TestOutOfOrderWithinWindow(t *testing.T) {
	f := &Filter{}
	for _, seq := range []uint16{1002, 1000, 1003, 1001} {
		if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: seq}) {
			t.Fatalf("out-of-order seq %d rejected", seq)
		}
	}
}

func TestOutOfOrderAcrossWrap(t *testing.T) {
	f := &Filter{}
	for _, seq := range []uint16{2, 65535, 0, 1} {
		if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: seq}) {
			t.Fatalf("seq %d rejected around wrap", seq)
		}
	}
}

func TestDuplicateRejected(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 500}) {
		t.Fatal("first packet rejected")
	}
	if f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 500}) {
		t.Fatal("duplicate of current max accepted")
	}
	f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 502})
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 501}) {
		t.Fatal("out-of-order packet rejected")
	}
	if f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 501}) {
		t.Fatal("duplicate of out-of-order packet accepted")
	}
}

func TestWindowBoundary(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 1000}) {
		t.Fatal("first packet rejected")
	}
	// A never-seen sequence 63 positions behind the max is inside the window.
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 1000 - 63}) {
		t.Fatal("packet 63 behind max rejected")
	}
	// 64 positions behind the max is outside the window.
	if f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 1000 - 64}) {
		t.Fatal("packet 64 behind max accepted")
	}
}

func TestTooOldAfterWindowSlides(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 1000}) {
		t.Fatal("first packet rejected")
	}
	for i := 1; i < windowSize; i++ {
		if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: uint16(1000 + i)}) {
			t.Fatalf("seq %d rejected", 1000+i)
		}
	}
	// The window now spans 1001..1064; 1000 has fallen out.
	if f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 1000}) {
		t.Fatal("packet outside slid window accepted")
	}
}

func TestNewerEpochResetsWindow(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 10}) {
		t.Fatal("first packet rejected")
	}
	if !f.Accept(Packet{Speaker: "a", Epoch: 2, Seq: 1}) {
		t.Fatal("newer epoch packet rejected")
	}
	// The empty epoch-2 window accepts a sequence the epoch-1 window had seen.
	if !f.Accept(Packet{Speaker: "a", Epoch: 2, Seq: 10}) {
		t.Fatal("sequence 10 rejected in fresh epoch")
	}
}

func TestOlderEpochStale(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 5, Seq: 10}) {
		t.Fatal("first packet rejected")
	}
	if f.Accept(Packet{Speaker: "a", Epoch: 4, Seq: 999}) {
		t.Fatal("stale epoch packet accepted")
	}
	// The stale packet must not disturb the current epoch's window.
	if !f.Accept(Packet{Speaker: "a", Epoch: 5, Seq: 11}) {
		t.Fatal("current epoch packet rejected after stale epoch packet")
	}
}

func TestSpeakersDoNotShareState(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 100}) {
		t.Fatal("speaker a first packet rejected")
	}
	// Same seq, same epoch, different speaker: independent first packet.
	if !f.Accept(Packet{Speaker: "b", Epoch: 1, Seq: 100}) {
		t.Fatal("speaker b packet rejected due to speaker a state")
	}
	// Speaker b's duplicate is rejected, speaker a's stream is unaffected.
	if f.Accept(Packet{Speaker: "b", Epoch: 1, Seq: 100}) {
		t.Fatal("speaker b duplicate accepted")
	}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 101}) {
		t.Fatal("speaker a stream broken by speaker b")
	}
	// A newer epoch for b must not reset a's window.
	if !f.Accept(Packet{Speaker: "b", Epoch: 2, Seq: 1}) {
		t.Fatal("speaker b newer epoch rejected")
	}
	if f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 100}) {
		t.Fatal("speaker a duplicate accepted after speaker b epoch change")
	}
}

func TestWorkBoundedOverLongStream(t *testing.T) {
	f := &Filter{}
	// 5000 sequential packets, crossing the sequence wrap.
	for i := 0; i < 5000; i++ {
		if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: uint16(i)}) {
			t.Fatalf("seq %d rejected in long stream", i)
		}
	}
	if f.Work > 6000 {
		t.Fatalf("Work = %d, want <= 6000", f.Work)
	}
}

func TestWorkBoundedWithDuplicatesAndEpochs(t *testing.T) {
	f := &Filter{}
	for i := 0; i < 5000; i++ {
		seq := uint16(i % 128)
		epoch := uint32(i / 1000)
		f.Accept(Packet{Speaker: "a", Epoch: epoch, Seq: seq})
		f.Accept(Packet{Speaker: "b", Epoch: epoch, Seq: seq})
	}
	if f.Work > 6000 {
		t.Fatalf("Work = %d, want <= 6000", f.Work)
	}
}
