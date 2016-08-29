package main

import (
	"fmt"

	"github.com/relab/byzq/proto/byzq"
)

// defaultVal is returned by the register when no quorum is reached.
// It is considered safe to return this value, rather than returning
// a reply that does not constitute a quorum.
var defaultVal = byzq.State{Value: -1, Timestamp: -1}

// ByzQ todo(doc) does something useful?
type ByzQ struct {
	wts int64 // write timestamp
	n   int   // size of system
	f   int   // tolerable number of failures
	q   int   // quorum size
}

// NewByzQ returns a Byzantine masking quorum specification or nil and an error
// if the quorum requirements are not satisifed.
func NewByzQ(n int) (*ByzQ, error) {
	f := (n - 1) / 4
	if f < 1 {
		return nil, fmt.Errorf("Byzantine masking quorums require n>4f replicas; only got n=%d, yielding f=%d", n, f)
	}
	return &ByzQ{0, n, f, (n + 2*f) / 2}, nil
}

// todo(meling) this wts is only suitable for single writer registers; multiple writers could perhaps be supported if wts was a combination of pid and wts?
func (bq *ByzQ) newWrite(val int64) *byzq.State {
	bq.wts++ //todo(meling) this needs a mutex lock (maybe??)
	return &byzq.State{Timestamp: bq.wts, Value: val}
}

// ReadQF returns nil and false until the supplied replies
// constitute a Byzantine masking quorum, at which point the
// method returns a single state and true.
func (bq *ByzQ) ReadQF(replies []*byzq.State) (*byzq.State, bool) {
	if len(replies) <= bq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}
	// filter out highest val that appears at least f times
	same := make(map[byzq.State]int)
	highest := defaultVal
	for _, reply := range replies {
		same[*reply]++
		// select reply with highest timestamp if it has more than f replies
		if same[*reply] > bq.f && reply.Timestamp > highest.Timestamp {
			highest = *reply
		}
	}
	// returns the reply with the highest timestamp, or if no quorum for
	// the same timestamp-value pair has been found, the defaultVal is returned.
	return &highest, true
}

// WriteQF returns nil and false until the supplied replies
// constitute a Byzantine masking quorum, at which point the
// method returns a single write response and true.
func (bq *ByzQ) WriteQF(replies []*byzq.WriteResponse) (*byzq.WriteResponse, bool) {
	if len(replies) <= bq.q {
		return nil, false
	}
	for _, ack := range replies {
		if bq.wts != ack.Timestamp {
			return nil, false
		}
	}
	return replies[0], true
}
