package packetepoch

// Packet is an inbound voice packet from one speaker.
type Packet struct {
	Speaker string
	Epoch   uint32
	Seq     uint16
}

// Filter decides whether a packet should enter playout.
type Filter struct {
	seen []Packet
	max  uint16
	Work int
}

func (f *Filter) Accept(p Packet) bool {
	for _, old := range f.seen {
		f.Work++
		if old.Seq == p.Seq {
			return false
		}
	}
	if len(f.seen) > 0 && p.Seq <= f.max {
		return false
	}
	f.seen = append(f.seen, p)
	f.max = p.Seq
	return true
}
