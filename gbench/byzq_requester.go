package gbench

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"strings"
	"time"

	rpc "github.com/relab/byzq"
	"github.com/tylertreat/bench"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ByzqRequesterFactory implements RequesterFactory by creating a Requester which
// issues requests to a storage using the Byzq framework.
type ByzqRequesterFactory struct {
	Addrs             []string
	PayloadSize       int
	QCTimeout         time.Duration
	WriteRatioPercent int
	NoAuth            bool
}

// GetRequester returns a new Requester, called for each Benchmark connection.
func (r *ByzqRequesterFactory) GetRequester(uint64) bench.Requester {
	return &byzqRequester{
		addrs:       r.Addrs,
		payloadSize: r.PayloadSize,
		timeout:     r.QCTimeout,
		writeRatio:  r.WriteRatioPercent,
		noauth:      r.NoAuth,
	}
}

type byzqRequester struct {
	addrs       []string
	payloadSize int
	timeout     time.Duration
	writeRatio  int
	noauth      bool

	mgr    *rpc.Manager
	config *rpc.Configuration
	qspec  *rpc.AuthDataQ
	state  *rpc.Content
}

const (
	serverCRT = `-----BEGIN CERTIFICATE-----
MIIEODCCAiCgAwIBAgIRAOVXpXvHjS/0TzD8VYkpnxIwDQYJKoZIhvcNAQELBQAw
FTETMBEGA1UEAxMKZ29ydW1zLm9yZzAeFw0xNzA5MTIyMjI0NDBaFw0xOTA5MTIy
MjI0NDBaMBQxEjAQBgNVBAMTCTEyNy4wLjAuMTCCASIwDQYJKoZIhvcNAQEBBQAD
ggEPADCCAQoCggEBAJvtFrfsu5bHOK7TkTHymHBClnSZKqlnlZkrrizOjGYDPr1w
t1MeEoKnjBQfSiuSsIzhpIfsza7EoQqpV3hWmLSg7RkKLyeGG9b1pbbyaLZDyljt
3ozLFfPh1oubwQtVAdVqJfIScfTv8tm/KUNg2nkMxshesG/gOm/BEKOdioSASmMS
SD6Xqz187TxTHyq4jSK1zR7E2D1plxyEa+xf1wtZqKBHVz5DiWWSVOnWuqdI+VFy
mEZgDzD3an+2KN/HFF3zq1ciRwp/fh7I0B1MjrbNEP5dgKrVtI8tSSIU731VUm8W
0EmnDToNKx2L0cL3l9I1pjKVI2jTXYZkmhSVMskCAwEAAaOBgzCBgDAOBgNVHQ8B
Af8EBAMCA7gwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMB0GA1UdDgQW
BBQ0DvmfxNXQ7L3tRtCkM70VeXYmPDAfBgNVHSMEGDAWgBQZCLlzr+Uc3rdSCUR4
9pBIcoHe3jAPBgNVHREECDAGhwR/AAABMA0GCSqGSIb3DQEBCwUAA4ICAQCg8ybL
5wTLZkmTHaAHX4cXAEYIKdr1hupAJrjLh8fKY7EzSI0b6CCgBX9jzEz76WcHCTJD
jY3fm+uYBbsB6ToCzqjv0bPtD3RyEVj7oDu+YsXIWLWeMDEGeIUy2qylNqKVJbKS
xaghlg60simzJtR2mQm/SmAIfcU+8kDyAYv1l2p+mvzHgryQPTybh6xKGeSpoC1o
JxtR1LhIPHih0C3FgC2W/nUAKKe/Lv6Zt+EDGJ/Yk9X80gP4v5RPytphflMudodn
Vvr0ZjBAc+blhRwTMVzxHJoJzwuCE8rYZWxjI+3cFNzGxls8IdphHrOepSlqXXHd
0rBdHBwCg4Nr/AYR6/HRG73E9dDGWw8WqRu2bOyqPognPxD/m9J24ocAumfT4e/8
7Weo2Z+5uJ2NUgERUl3SzEBWMO7s1NuqUkn7Tnmi5DAGiZr/Ykrot/mLUCsxRsOR
HPreqRWk7bSDtHCmKUnGUj3t6Uw6+GFEfuMwl4n4Dr9XGRmLl8tAyWU/Q87MU+Yy
Qj3E91GCrKqWgqpspSepz/nOU3ZkaMPYcp98ZAehYWBF5Dd/Y/r5Oo8HwnwHta3y
Hi+MnWeLAwM+yGxnIftBDm2PDMpfNksVbOaCFZit1GEG7dz0bSTZh9oJtSXucAKs
xcnHJWVF35BQqMaNKccUnxUM7x8ujle/GgB7Hw==
-----END CERTIFICATE-----`

	keyPEM = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIANyDBAupB6O86ORJ1u95Cz6C+lz3x2WKOFntJNIesvioAoGCCqGSM49
AwEHoUQDQgAE+pBXRIe0CI3vcdJwSvU37RoTqlPqEve3fcC36f0pY/X9c9CsgkFK
/sHuBztq9TlUfC0REC81NRqRgs6DTYJ/4Q==
-----END EC PRIVATE KEY-----`
)

func (gr *byzqRequester) Setup() error {
	var secDialOption grpc.DialOption
	if gr.noauth {
		secDialOption = grpc.WithInsecure()
	} else {
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM([]byte(serverCRT)) {
			return fmt.Errorf("credentials: failed to append certificates")
		}
		clientCreds := credentials.NewTLS(&tls.Config{ServerName: "127.0.0.1", RootCAs: cp})
		secDialOption = grpc.WithTransportCredentials(clientCreds)
	}
	var err error
	gr.mgr, err = rpc.NewManager(
		gr.addrs,
		rpc.WithGrpcDialOptions(
			grpc.WithBlock(),
			grpc.WithTimeout(time.Second),
			secDialOption,
		),
	)
	if err != nil {
		return err
	}

	key, err := rpc.ParseKey(keyPEM)
	if err != nil {
		return err
	}

	ids := gr.mgr.NodeIDs()
	gr.qspec, err = rpc.NewAuthDataQ(len(ids), key, &key.PublicKey)
	if err != nil {
		return err
	}

	gr.config, err = gr.mgr.NewConfiguration(ids, gr.qspec)
	if err != nil {
		return err
	}

	// Set initial state.
	gr.state = &rpc.Content{
		Key:       "State",
		Value:     strings.Repeat("x", gr.payloadSize),
		Timestamp: time.Now().UnixNano(),
	}
	// Sign initial state
	signedState, err := gr.qspec.Sign(gr.state)
	if err != nil {
		return err
	}
	ack, err := gr.config.Write(context.Background(), signedState)
	if err != nil {
		return fmt.Errorf("write rpc error: %v", err)
	}
	if ack.Timestamp == 0 {
		return fmt.Errorf("intital write reply was not marked as new")
	}
	return nil
}

func (gr *byzqRequester) Request() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), gr.timeout)
	defer cancel()
	switch gr.writeRatio {
	case 0:
		_, err = gr.config.Read(ctx, &rpc.Key{Key: gr.state.Key})
	case 100:
		gr.state.Timestamp = time.Now().UnixNano()
		signedState, err2 := gr.qspec.Sign(gr.state)
		if err2 != nil {
			return err2
		}
		_, err = gr.config.Write(ctx, signedState)
	default:
		x := rand.Intn(100)
		if x < gr.writeRatio {
			gr.state.Timestamp = time.Now().UnixNano()
			signedState, err2 := gr.qspec.Sign(gr.state)
			if err2 != nil {
				return err
			}
			_, err = gr.config.Write(ctx, signedState)
		} else {
			_, err = gr.config.Read(ctx, &rpc.Key{Key: gr.state.Key})
		}
	}

	return err
}

func (gr *byzqRequester) Teardown() error {
	gr.mgr.Close()
	gr.mgr = nil
	gr.config = nil
	return nil
}
