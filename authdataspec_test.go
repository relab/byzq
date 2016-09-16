package byzq

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"
)

// run tests with: go test -v

// run benchmarks: go test -run=$$ -benchmem -benchtime=5s -bench=.

// func TestMain(m *testing.M) {
// 	silentLogger := log.New(ioutil.Discard, "", log.LstdFlags)
// 	grpclog.SetLogger(silentLogger)
// 	grpc.EnableTracing = false
// 	res := m.Run()
// 	os.Exit(res)
// }

var priv, _ = readKeyfile()

var pemKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIANyDBAupB6O86ORJ1u95Cz6C+lz3x2WKOFntJNIesvioAoGCCqGSM49
AwEHoUQDQgAE+pBXRIe0CI3vcdJwSvU37RoTqlPqEve3fcC36f0pY/X9c9CsgkFK
/sHuBztq9TlUfC0REC81NRqRgs6DTYJ/4Q==
-----END EC PRIVATE KEY-----`

func readKeyfile() (*ecdsa.PrivateKey, error) {
	// Crypto (TODO clean up later)
	// See https://golang.org/src/crypto/tls/generate_cert.go
	key := new(ecdsa.PrivateKey)

	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("no block to decode")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key from pem block: %v\n %v", err, key)
	}
	return key, nil
}

var authQTests = []struct {
	n   int
	f   int // expected value
	q   int // expected value
	err string
}{
	{3, 0, 2, "Byzantine quorum require n>3f replicas; only got n=3, yielding f=0"},
	{4, 1, 2, ""},
	{5, 1, 3, ""},
	{6, 1, 3, ""},
	{7, 2, 4, ""},
	{8, 2, 5, ""},
	{9, 2, 5, ""},
	{10, 3, 6, ""},
	{11, 3, 7, ""},
	{12, 3, 7, ""},
	{13, 4, 8, ""},
	{14, 4, 9, ""},
}

func TestAuthDataQ(t *testing.T) {
	for _, test := range authQTests {
		bq, err := NewAuthDataQ(test.n, priv, &priv.PublicKey)
		if err != nil {
			if err.Error() != test.err {
				t.Errorf("got '%v', expected '%v'", err.Error(), test.err)
			}
			continue
		}
		if bq.f != test.f {
			t.Errorf("got f=%d, expected f=%d", bq.f, test.f)
		}
		if bq.q != test.q {
			t.Errorf("got q=%d, expected q=%d", bq.q, test.q)
		}
	}

}

var (
	myContent = &Content{Key: "Winnie", Value: "Poo", Timestamp: 3}
	myVal     = &Value{C: myContent}
)

var authReadQFTests = []struct {
	name     string
	replies  []*Value
	expected *Value
	rq       bool
}{
	{
		"nil input",
		nil,
		nil,
		false,
	},
	{
		"len=0 input",
		[]*Value{},
		nil,
		false,
	},
	{
		"no quorum (I)",
		[]*Value{
			&Value{C: &Content{Key: "winnie", Value: "2", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: myVal.C.Value, Timestamp: 1}},
		},
		nil,
		false,
	},
	{
		"no quorum (II) not enough equal replies",
		[]*Value{
			&Value{C: &Content{Key: "winnie", Value: "2", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: "3", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: myVal.C.Value, Timestamp: 1}},
		},
		nil,
		false,
	},
	{
		"no quorum (III); not enough equal replies",
		[]*Value{
			&Value{C: &Content{Key: "winnie", Value: "2", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: "3", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: "4", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: myVal.C.Value, Timestamp: 1}},
		},
		nil,
		false,
	},
	{
		"quorum (I)",
		[]*Value{
			myVal,
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"quorum (II)",
		[]*Value{
			&Value{C: &Content{Key: "winnie", Value: "2", Timestamp: 1}},
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"quorum (III)",
		[]*Value{
			myVal,
			myVal,
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"quorum (IV)",
		[]*Value{
			&Value{C: &Content{Key: "winnie", Value: "2", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: "2", Timestamp: 1}},
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"base-case quorum",
		[]*Value{
			myVal,
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"approx. worst-case quorum",
		[]*Value{
			&Value{C: &Content{Key: "winnie", Value: "2", Timestamp: 1}},
			&Value{C: &Content{Key: "winnie", Value: "4", Timestamp: 2}},
			&Value{C: &Content{Key: "winnie", Value: "5", Timestamp: 1}},
			myVal,
			myVal,
		},
		myVal,
		true,
	},
}

func TestAuthDataReadQF(t *testing.T) {
	qspec, err := NewAuthDataQ(4, priv, &priv.PublicKey)
	if err != nil {
		t.Error(err)
	}
	for _, test := range authReadQFTests {
		for i, r := range test.replies {
			test.replies[i], err = qspec.Sign(r.C)
			if err != nil {
				t.Fatal("Failed to sign message")
			}
		}
		t.Run("AuthDataQ(4,1)-"+test.name, func(t *testing.T) {
			reply, byzquorum := qspec.ReadQF(test.replies)
			if byzquorum != test.rq {
				t.Errorf("got %t, want %t", byzquorum, test.rq)
			}
			if !reply.Equal(test.expected) {
				t.Errorf("got %v, want %v as quorum reply", reply, test.expected)
			}
		})
	}
}

func BenchmarkAuthDataReadQF(b *testing.B) {
	qspec, err := NewAuthDataQ(4, priv, &priv.PublicKey)
	if err != nil {
		b.Error(err)
	}
	for _, test := range authReadQFTests {
		if !strings.Contains(test.name, "case") {
			continue
		}
		for i, r := range test.replies {
			test.replies[i], err = qspec.Sign(r.C)
			if err != nil {
				b.Fatal("Failed to sign message")
			}
		}
		b.Run("AuthDataQ(4,1)-"+test.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				qspec.ReadQF(test.replies)
			}
		})
	}
}
