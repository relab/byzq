package byzq

import (
	math "math"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestAuthDataQSpecProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("undefined for all n <= 3", prop.ForAll(
		func(n int) bool {
			bq, err := NewAuthDataQ(n, priv, &priv.PublicKey)
			return err != nil && bq == nil
		},
		gen.IntRange(math.MinInt32, 3),
	))

	properties.Property("3(f+1) >= n >= 3f+1", prop.ForAll(
		func(n int) bool {
			bq, err := NewAuthDataQ(n, priv, &priv.PublicKey)
			return err == nil && n >= 3*bq.f+1 && n <= 3*(bq.f+1)
		},
		gen.IntRange(4, math.MaxInt32),
	))

	properties.TestingRun(t)
}

func TestAuthDataQuorumProperties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("no quorum unless enough replies", prop.ForAll(
		func(n int) bool {
			qspec, err := NewAuthDataQ(n, priv, &priv.PublicKey)
			if err != nil {
				return false
			}
			nonQuormSize, ok := gen.IntRange(0, qspec.q).Sample()
			if !ok {
				return false
			}
			sliceGen := gen.SliceOfN(nonQuormSize.(int), gen.Const(myVal))
			result := sliceGen(gopter.DefaultGenParameters())
			value, ok := result.Retrieve()
			if !ok || value == nil {
				return false
			}
			replies, ok := value.([]*Value)
			if !ok || len(replies) != nonQuormSize {
				return false
			}
			reply, byzquorum := qspec.ReadQF(replies)
			return !byzquorum && reply == nil
		},
		gen.IntRange(4, 200),
	))

	properties.Property("sufficient replies guarantees a quorum", prop.ForAll(
		func(n int) bool {
			qspec, err := NewAuthDataQ(n, priv, &priv.PublicKey)
			if err != nil {
				return false
			}
			numReplies, ok := gen.IntRange(qspec.q+1, qspec.n).Sample()
			if !ok {
				return false
			}
			sliceGen := gen.SliceOfN(numReplies.(int), gen.Const(myVal))
			result := sliceGen(gopter.DefaultGenParameters())
			value, ok := result.Retrieve()
			if !ok || value == nil {
				t.Errorf("invalid value: %#v", value)
				return false
			}
			replies, ok := value.([]*Value)
			if !ok || len(replies) != numReplies {
				t.Errorf("invalid value: %#v", value)
				return false
			}
			for i, r := range replies {
				replies[i], err = qspec.Sign(r.C)
				if err != nil {
					t.Fatal("failed to sign message")
				}
			}
			reply, byzquorum := qspec.ReadQF(replies)
			if !byzquorum {
				return false
			}
			for _, r := range replies {
				if reply.Equal(r.GetC()) {
					return true
				}
			}
			return false
		},
		gen.IntRange(4, 200),
	))

	properties.TestingRun(t)
}

// func QuorumRange(n, lower, upper int) gopter.Gen {
// 	qspec, err := NewAuthDataQ(n, priv, &priv.PublicKey)
// 	if err != nil {
// 		return gen.Fail(nil)
// 	}
// 	numReplies, _ := gen.IntRange(qspec.q+1, qspec.n).Sample()

// }
