package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"sync"

	"github.com/relab/byzq"
)

// AuthDataQ todo(doc) does something useful?
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

// PreWrite is invoked before Write()
// TODO debug why this does not return a Value out (println after calling PreWrite)
func (aq *AuthDataQ) PreWrite(args byzq.Value) (err error) {
	a, err := aq.Sign(args.C)
	args = *a
	fmt.Println(a.SignatureR)
	return
}

// Sign signs the provided content and returns a value to be passed into Write.
func (aq *AuthDataQ) Sign(content *byzq.Content) (*byzq.Value, error) {
	msg, err := content.Marshal()
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(msg)
	fmt.Println(hash)
	r, s, err := ecdsa.Sign(rand.Reader, aq.priv, hash[:])
	if err != nil {
		return nil, err
	}
	return &byzq.Value{C: content, SignatureR: r.Bytes(), SignatureS: s.Bytes()}, nil
}

func (aq *AuthDataQ) verify(reply *byzq.Value, index int, resultchan chan int) bool {
	// TODO add Byzantine behavior by changing return value and detect verify failure.
	msg, err := reply.C.Marshal()
	if err != nil {
		//FIXME log error
		// dief("failed to marshal msg for verify: %v", err)
		resultchan <- -1
		return false
	}
	fmt.Println("content = ", reply.C.String())
	fmt.Println("msg = ", msg)
	msgHash := sha256.Sum256(msg)
	r := new(big.Int).SetBytes(reply.SignatureR)
	s := new(big.Int).SetBytes(reply.SignatureS)
	// s.Add(s, one) // Byzantine behavior (add 1 to signature field)

	//TODO make this return directly
	if !ecdsa.Verify(&aq.priv.PublicKey, msgHash[:], r, s) {
		//FIXME log error
		fmt.Println("couldn't verify signature: ") // + val.String())
		fmt.Println("msgHash = ", msgHash)
		fmt.Println("r = ", r)
		fmt.Println("s = ", s)
		resultchan <- -1
		return false
	}
	resultchan <- index
	return true
}

// ReadQF returns nil and false until the supplied replies
// constitute a Byzantine masking quorum, at which point the
// method returns a single state and true.
func (aq *AuthDataQ) ReadQF(replies []*byzq.Value) (*byzq.Value, bool) {
	//TODO could also verify reply in goroutine here and also fire up a collection goroutine that keep track of things

	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}
	// filter out highest val that appears at least f times
	same := make(map[byzq.Content]int)
	highest := defaultVal
	for _, reply := range replies {
		// TODO use goroutines to speed up this verification
		if aq.verify(reply) {
			same[*reply.C]++
			// select reply with highest timestamp if it has more than f replies
			if same[*reply.C] > aq.f && reply.C.Timestamp > highest.C.Timestamp {
				highest = *reply
			}
		}
	}

	//TODO Need to return nil, false if not enough correct replies received (not defaultVal)

	// returns the reply with the highest timestamp, or if no quorum for
	// the same timestamp-value pair has been found, the defaultVal is returned.
	return &highest, true
}

//Leanders version og QFunc
func (aq *AuthDataQ) LReadQF(replies []*byzq.Value) (*byzq.Value, bool) {
	//TODO could also verify reply in goroutine here and also fire up a collection goroutine that keep track of things

	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}

	veriresult := new(chan int, len(replies))

	for i, reply := range replies {
		go aq.verify(reply,i,veriresult) 
	}

	cnt := 0
	for {
		i:= <-veriresult:
	}

	if len(replies)-cnt <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}

	// filter out highest val that appears at least f times
	//same := make(map[byzq.Content]int)
	highest := &defaultVal
	wg.Wait()

	for _, reply := range replies {
		if reply == nil {
			continue
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

//Leanders version og QFunc Version2
func (aq *AuthDataQ) L2ReadQF(replies []*byzq.Value) (*byzq.Value, bool) {
	//TODO could also verify reply in goroutine here and also fire up a collection goroutine that keep track of things

	// Continue if last reply does not verify.
	if !aq.verify(replies[len(replies)-1]) {
		replies[len(replies)-1] = nil
		return nil, false
	}

	if len(replies) <= aq.q {
		// not enough replies yet; need at least bq.q=(n+2f)/2 replies
		return nil, false
	}

	// filter out highest val that appears at least f times
	//same := make(map[byzq.Content]int)
	highest := &defaultVal

	cntnotnil := 0
	for _, reply := range replies {
		if reply == nil {
			continue
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
func (aq *AuthDataQ) WriteQF(replies []*byzq.WriteResponse) (*byzq.WriteResponse, bool) {
	if len(replies) <= aq.q {
		return nil, false
	}
	return replies[0], true
}
