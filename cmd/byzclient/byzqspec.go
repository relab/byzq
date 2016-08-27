package main

import "github.com/relab/byzq/proto/byzq"

// defaultVal is returned by the register when no quorum is reached.
// It is considered safe to return this value, rather than returning
// a reply that does not constitute a quorum.
var defaultVal = &byzq.State{Value: -1, Timestamp: -1}

// ByzQ todo(doc) does something useful?
type ByzQ struct {
	wts int // write timestamp
	n   int // size of system
	f   int // tolerable number of failures
	q   int // quorum size
}

// NewByzQ returns a Byzantine masking quorum specification.
func NewByzQ(n, f int) *ByzQ {
	//todo(meling) should return error if n too low to satisfy f.
	return &ByzQ{0, n, f, (n + 2*f) / 2}
}

// ReadQF returns nil and false until the supplied replies
// constitute a Byzantine masking quorum, at which point the
// method returns a single state and true.
func (bq *ByzQ) ReadQF(replies []*byzq.State) (*byzq.State, bool) {
	if len(replies) <= bq.q {
		return nil, false
	}
	// filter out highest val that appears at least f times
	same := make(map[byzq.State]int)
	for _, reply := range replies {
		same[*reply]++
	}
	highest := defaultVal
	for reply, count := range same {
		// select reply with highest timestamp if it has more than f replies
		if count > bq.f && reply.Timestamp > highest.Timestamp {
			highest = &reply
		}
	}
	// returns the reply with the highest timestamp, or if no quorum for
	// the same timestamp-value pair has been found, the defaultVal is returned.
	return highest, true
}

// WriteQF returns nil and false until the supplied replies
// constitute a Byzantine masking quorum, at which point the
// method returns a single write response and true.
func (bq *ByzQ) WriteQF(replies []*byzq.WriteResponse) (*byzq.WriteResponse, bool) {
	if len(replies) <= bq.q {
		// fmt.Println("reply len()=" + strconv.Itoa(len(replies)))
		return nil, false
	}

	return replies[0], true
}
