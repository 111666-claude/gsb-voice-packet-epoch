package packetepoch

import "testing"

func TestFirstPacket(t *testing.T) {
	f := &Filter{}
	if !f.Accept(Packet{Speaker: "a", Epoch: 1, Seq: 10}) {
		t.Fatal("first packet rejected")
	}
}
