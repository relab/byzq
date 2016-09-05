package main

import (
	"fmt"

	"github.com/relab/byzq/proto/byzq"
)

// AuthDataQ todo(doc) does something useful?
type AuthDataQ struct {
	n int // size of system
	f int // tolerable number of failures
	q int // quorum size
}

// NewAuthDataQ returns a Byzantine masking quorum specification or nil and an error
// if the quorum requirements are not satisifed.
func NewAuthDataQ(n int) (*AuthDataQ, error) {
	f := (n - 1) / 3
	if f < 1 {
		return nil, fmt.Errorf("Byzantine quorum require n>3f replicas; only got n=%d, yielding f=%d", n, f)
	}
	return &AuthDataQ{n, f, (n + f) / 2}, nil
}

// ReadQF returns nil and false until the supplied replies
// constitute a Byzantine masking quorum, at which point the
// method returns a single state and true.
func (aq *AuthDataQ) ReadQF(replies []*byzq.State) (*byzq.State, bool) {
	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}
	// filter out highest val that appears at least f times
	same := make(map[byzq.State]int)
	highest := defaultVal
	for _, reply := range replies {
		same[*reply]++
		// select reply with highest timestamp if it has more than f replies
		if same[*reply] > aq.f && reply.Timestamp > highest.Timestamp {
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
func (aq *AuthDataQ) WriteQF(replies []*byzq.WriteResponse) (*byzq.WriteResponse, bool) {
	if len(replies) <= aq.q {
		return nil, false
	}
	return replies[0], true
}
