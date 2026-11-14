# Voice packet replay window

Voice packets carry a speaker ID, an epoch and a wrapping 16-bit sequence
number. Each speaker has an independent 64-packet acceptance window. A newer
epoch resets that speaker's window; an older epoch is stale. Sequence numbers
wrap from 65535 to 0. A packet within the current window may arrive out of
order and should be accepted once; a duplicate or packet outside the window
must be rejected. A speaker must not affect another speaker's state.

`Work` counts historical packet entries examined during acceptance. A stream
of many packets must keep per-packet work bounded by the 64-packet window,
including after sequence wrap and epoch changes.
