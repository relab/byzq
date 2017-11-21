package byzq

import (
	"math"
	"reflect"
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

	replyGen := func(numReplies int) []*Value {
		sliceGen := gen.SliceOfN(numReplies, gen.Const(myVal))
		result := sliceGen(gopter.DefaultGenParameters())
		value, ok := result.Retrieve()
		if !ok || value == nil {
			t.Fatalf("invalid value: %#v", value)
		}
		replies, ok := value.([]*Value)
		if !ok || len(replies) != numReplies {
			t.Fatalf("invalid number of replies: %d, expected: %d", len(replies), numReplies)
		}
		return replies
	}

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
			replies := replyGen(nonQuormSize.(int))
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
			replies := replyGen(numReplies.(int))
			for i, r := range replies {
				replies[i], err = qspec.Sign(r.C)
				if err != nil {
					t.Fatal("failed to sign message")
				}
			}
			reply, byzquorum := qspec.ConcurrentVerifyWGReadQF(replies)
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

	type qfParams struct {
		quorumSize int
		qspec      *AuthDataQ
	}

	properties.Property("sufficient replies guarantees a quorum", prop.ForAll(
		func(params *qfParams) bool {
			replies := replyGen(params.quorumSize)
			var err error
			for i, r := range replies {
				replies[i], err = params.qspec.Sign(r.C)
				if err != nil {
					t.Fatal("failed to sign message")
				}
			}
			reply, byzquorum := params.qspec.SequentialVerifyReadQF(replies)
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
		gen.IntRange(4, 100).FlatMap(func(n interface{}) gopter.Gen {
			qspec, err := NewAuthDataQ(n.(int), priv, &priv.PublicKey)
			if err != nil {
				t.Fatalf("failed to create quorum specification for size %d", n)
			}
			return gen.IntRange(qspec.q+1, qspec.n).Map(func(quorumSize interface{}) *qfParams {
				return &qfParams{quorumSize.(int), qspec}
			})
		}, reflect.TypeOf(&qfParams{})),
	))

	properties.TestingRun(t)
}
