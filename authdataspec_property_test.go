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

	properties.Property("Undefined for all <= 0", prop.ForAll(
		func(n int) bool {
			bq, err := NewAuthDataQ(n, priv, &priv.PublicKey)
			return err != nil && bq == nil
		},
		gen.IntRange(math.MinInt32, 0),
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
