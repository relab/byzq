package byzq

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"math/big"
)

// AuthDataQ is the quorum specification for the Authenticated-Data Byzantine quorum
// algorithm described in TODO
type AuthDataQ struct {
	n    int               // size of system
	f    int               // tolerable number of failures
	q    int               // quorum size
	priv *ecdsa.PrivateKey // my private key for signing
	pubk *ecdsa.PublicKey  // map of public keys of other writers/signers (clients)
}

// NewAuthDataQ returns a Byzantine masking quorum specification or nil and an error
// if the quorum requirements are not satisifed.
func NewAuthDataQ(n int, priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey) (*AuthDataQ, error) {
	f := (n - 1) / 3
	if f < 1 {
		return nil, fmt.Errorf("Byzantine quorum require n>3f replicas; only got n=%d, yielding f=%d", n, f)
	}
	return &AuthDataQ{n, f, (n + f) / 2, priv, pub}, nil
}

// Sign signs the provided content and returns a value to be passed into Write.
func (aq *AuthDataQ) Sign(content *Content) (*Value, error) {
	msg, err := content.Marshal()
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(msg)
	r, s, err := ecdsa.Sign(rand.Reader, aq.priv, hash[:])
	if err != nil {
		return nil, err
	}
	return &Value{C: content, SignatureR: r.Bytes(), SignatureS: s.Bytes()}, nil
}

func (aq *AuthDataQ) verify(reply *Value) bool {
	msg, err := reply.C.Marshal()
	if err != nil {
		log.Printf("failed to marshal msg for verify: %v", err)
		return false
	}
	msgHash := sha256.Sum256(msg)
	r := new(big.Int).SetBytes(reply.SignatureR)
	s := new(big.Int).SetBytes(reply.SignatureS)
	return ecdsa.Verify(&aq.priv.PublicKey, msgHash[:], r, s)
}

func (aq *AuthDataQ) pverify(reply *Value, index int, resultchan chan int) {
	msg, err := reply.C.Marshal()
	if err != nil {
		log.Printf("failed to marshal msg for verify: %v", err)
		resultchan <- -1
		return
	}
	msgHash := sha256.Sum256(msg)
	r := new(big.Int).SetBytes(reply.SignatureR)
	s := new(big.Int).SetBytes(reply.SignatureS)

	if !ecdsa.Verify(&aq.priv.PublicKey, msgHash[:], r, s) {
		log.Printf("failed to verify signature for: %v", reply.C)
		resultchan <- -1
		return
	}
	resultchan <- index
}

// ReadQF returns nil and false until the supplied replies
// constitute a Byzantine quorum, at which point the
// method returns the single highest value and true.
func (aq *AuthDataQ) ReadQF(replies []*Value) (*Value, bool) {
	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}
	var highest *Value
	for _, reply := range replies {
		if aq.verify(reply) {
			if highest != nil && reply.C.Timestamp <= highest.C.Timestamp {
				continue
			}
			highest = reply
		}
	}

	// returns reply with the highest timestamp, or nil if not enough
	// replies were verified
	return highest, true
}

func (aq *AuthDataQ) cverify(reply *Value, index int, resultchan chan int) {
	if aq.verify(reply) {
		resultchan <- index
	} else {
		resultchan <- -1
	}
}

// LReadQF returns Leanders QFunc version 1
func (aq *AuthDataQ) LReadQF(replies []*Value) (*Value, bool) {
	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}

	veriresult := make(chan int, len(replies))

	for i, reply := range replies {
		go aq.cverify(reply, i, veriresult)
	}

	cnt := 0
	var highest *Value
	for j := 0; j < len(replies); j++ {
		i := <-veriresult
		if i == -1 {
			//some signature could not be verified:
			cnt++
			if len(replies)-cnt <= aq.q {
				return nil, false
			}
		}
		if highest != nil && replies[i].C.Timestamp <= highest.C.Timestamp {
			continue
		}
		highest = replies[i]
	}

	return highest, true
}

// XLReadQF returns Leanders QFunc version 1old
func (aq *AuthDataQ) XLReadQF(replies []*Value) (*Value, bool) {
	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}

	veriresult := make(chan int, len(replies))
	for i, reply := range replies {
		go aq.pverify(reply, i, veriresult)
	}

	cnt := 0
	indicies := make([]int, len(replies))
	for i := range veriresult {
		indicies = append(indicies, i)
		cnt++
	}

	if len(replies)-cnt <= aq.q {
		// not enough verified replies yet; need at least bq.q=(n+2f)/2 correct replies
		return nil, false
	}

	// filter out highest val that appears at least f times
	var highest *Value
	// wg.Wait()

	for _, reply := range replies {
		if reply == nil {
			continue
		}
		if highest == nil {
			// only verified replies should be considered as highest
			highest = reply
		}

		// select reply with highest timestamp
		if reply.C.Timestamp > highest.C.Timestamp {
			highest = reply
		}
	}

	//TODO Need to return nil, false if not enough correct replies received (not defaultVal)

	// returns the reply with the highest timestamp, or if no quorum for
	// the same timestamp-value pair has been found, the defaultVal is returned.
	return highest, true
}

// L2ReadQF returns Leanders QFunc version 2
func (aq *AuthDataQ) L2ReadQF(replies []*Value) (*Value, bool) {
	if !aq.verify(replies[len(replies)-1]) {
		// Continue if last reply does not verify.
		replies[len(replies)-1] = nil
		return nil, false
	}

	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}

	// filter out highest val that appears at least f times
	//same := make(map[Content]int)
	var highest *Value

	cntnotnil := 0
	for _, reply := range replies {
		if reply == nil {
			continue
		}
		if highest == nil {
			// only verified replies should be considered as highest
			highest = reply
		}

		cntnotnil++
		// select reply with highest timestamp
		if reply.C.Timestamp > highest.C.Timestamp {
			highest = reply
		}
	}

	if cntnotnil <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}

	//TODO Need to return nil, false if not enough correct replies received (not defaultVal)

	// returns the reply with the highest timestamp, or if no quorum for
	// the same timestamp-value pair has been found, the defaultVal is returned.
	return highest, true
}

// WriteQF returns nil and false until the supplied replies
// constitute a Byzantine masking quorum, at which point the
// method returns a single write response and true.
func (aq *AuthDataQ) WriteQF(replies []*WriteResponse) (*WriteResponse, bool) {
	if len(replies) <= aq.q {
		return nil, false
	}
	return replies[0], true
}
