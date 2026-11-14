package packetepoch

// windowSize is the number of sequence positions in a speaker's acceptance
// window. Bit i of speakerState.seen records whether sequence
// (maxSeq - i) mod 2^16 has already been accepted.
const windowSize = 64

// halfSeqSpace splits the circular 16-bit sequence space into "ahead" and
// "behind" the current maximum.
const halfSeqSpace = 1 << 15

// Packet is an inbound voice packet from one speaker.
type Packet struct {
	Speaker string
	Epoch   uint32
	Seq     uint16
}

// Filter decides whether a packet should enter playout.
//
// Each speaker has independent state: its own epoch and its own 64-packet
// window. Sequence numbers are interpreted circularly, so the window slides
// correctly across the 65535 -> 0 wrap. Acceptance checks a single bitmap
// word, so the per-packet cost does not grow with the number of packets
// already seen.
type Filter struct {
	speakers map[string]*speakerState

	// Work counts historical packet entries examined during acceptance.
	// The bitmap representation examines none, so it stays constant.
	Work int
}

type speakerState struct {
	started bool
	epoch   uint32
	maxSeq  uint16
	// seen bit i is set when (maxSeq-i) mod 2^16 has been accepted.
	seen uint64
}

func (f *Filter) Accept(p Packet) bool {
	st := f.speaker(p.Speaker)

	if !st.started {
		st.start(p)
		return true
	}

	if p.Epoch != st.epoch {
		// A newer epoch resets this speaker's window; an older epoch is stale.
		if p.Epoch < st.epoch {
			return false
		}
		st.start(p)
		return true
	}

	if p.Seq == st.maxSeq {
		// Bit 0 is always set for the current maximum, so this is a duplicate.
		return false
	}

	if ahead := uint16(p.Seq - st.maxSeq); ahead < halfSeqSpace {
		// New packet ahead of the window: advance it, dropping entries that
		// fall outside the 64-packet span.
		if ahead >= windowSize {
			st.seen = 0
		} else {
			st.seen <<= ahead
		}
		st.seen |= 1
		st.maxSeq = p.Seq
		return true
	}

	behind := uint16(st.maxSeq - p.Seq)
	if behind >= windowSize {
		// Too old: falls outside the window.
		return false
	}
	bit := uint64(1) << behind
	if st.seen&bit != 0 {
		return false // duplicate within the window
	}
	st.seen |= bit
	return true
}

func (st *speakerState) start(p Packet) {
	st.started = true
	st.epoch = p.Epoch
	st.maxSeq = p.Seq
	st.seen = 1
}

func (f *Filter) speaker(id string) *speakerState {
	if f.speakers == nil {
		f.speakers = make(map[string]*speakerState)
	}
	st := f.speakers[id]
	if st == nil {
		st = &speakerState{}
		f.speakers[id] = st
	}
	return st
}
