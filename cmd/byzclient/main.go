package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/relab/byzq"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const localAddrs = ":8080,:8081,:8082,:8083,:8084"

var one = new(big.Int).SetInt64(1)

func main() {
	saddrs := flag.String("addrs", localAddrs, "server addresses separated by ','")
	writer := flag.Bool("writer", false, "set this client to be writer only (default is reader only)")
	protocol := flag.String("protocol", "byzq", "protocol to use in the experiment (options: byzq, authq)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	addrs := strings.Split(*saddrs, ",")
	if len(addrs) == 0 {
		dief("no server addresses provided")
	}
	log.Println("#addrs:", len(addrs))

	//TODO fix hardcoded youtube server name (can we get certificate for localhost servername?)
	clientCreds, err := credentials.NewClientTLSFromFile("cert/ca.pem", "x.test.youtube.com")
	if err != nil {
		dief("error creating credentials: %v", err)
	}

	mgr, err := byzq.NewManager(
		addrs,
		byzq.WithGrpcDialOptions(
			grpc.WithBlock(),
			grpc.WithTimeout(0*time.Millisecond),
			grpc.WithTransportCredentials(clientCreds),
		),
	)
	if err != nil {
		dief("error creating manager: %v", err)
	}
	defer mgr.Close()

	ids := mgr.NodeIDs()
	var qspec byzq.QuorumSpec
	switch *protocol {
	case "byzq":
		qspec, err = NewByzQ(len(ids))
		// case "authq":
		// 	qspec, err = NewAuthDataQ(len(ids))
	}
	if err != nil {
		dief("%v", err)
	}
	conf, err := mgr.NewConfiguration(ids, qspec, time.Second)
	if err != nil {
		dief("error creating config: %v", err)
	}

	content := &byzq.Content{
		Key:       "Hein",
		Value:     "Meling",
		Timestamp: -1,
	}

	keyFile := "my-key.pem"

	// Crypto (TODO clean up later)
	// See https://golang.org/src/crypto/tls/generate_cert.go
	key := new(ecdsa.PrivateKey)

	f, err := os.Open(keyFile)
	if err != nil {
		if !os.IsNotExist(err) {
			dief("failed to open key file: %v", err)
		}
		f, err = os.Create(keyFile)
		if err != nil {
			dief("failed to create key file: %v", err)
		}

		DefaultCurve := elliptic.P256()
		key, err = ecdsa.GenerateKey(DefaultCurve, crand.Reader)
		if err != nil {
			f.Close()
			dief("failed to generate keys: %v", err)
		}

		// keypem, err := os.OpenFile("ec-key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		ec, err := x509.MarshalECPrivateKey(key)
		pem.Encode(f, &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: ec,
		})

		// secp256r1, err := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7})
		// pem.Encode(f, &pem.Block{Type: "EC PARAMETERS", Bytes: secp256r1})

		if err != nil {
			f.Close()
			dief("failed to write key: %v\n %v", err, key)
		}
		f.Close()
	} else {
		b := make([]byte, 500)
		_, err := f.Read(b)
		// fmt.Println(n)
		// fmt.Println(b)
		if err != nil {
			f.Close()
			dief("failed to read key: %v", err)
		}
		f.Close()
		block, _ := pem.Decode(b)
		if block == nil {
			dief("no block to decode")
		}
		key, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			dief("failed to parse key from pem block: %v\n %v", err, key)
		}
	}

	// TODO(meling) Sign this content object (for the authq protocol)

	registerState := byzq.Value{C: content}
	for {
		if *writer {
			// Writer client
			k := rand.Intn(1 << 8)
			registerState.C.Value = strconv.Itoa(k)
			registerState.C.Timestamp++
			msg, err := registerState.C.Marshal()
			if err != nil {
				dief("failed to marshal msg for signing: %v", err)
			}
			fmt.Println("content = ", registerState.C.String())
			fmt.Println("msg = ", msg)
			msgHash := sha256.Sum256(msg)
			r, s, err := ecdsa.Sign(crand.Reader, key, msgHash[:])
			if err != nil {
				dief("failed to sign msg: %v", err)
			}
			fmt.Println("signature:")
			fmt.Println("msgHash = ", msgHash)
			fmt.Println("r = ", r)
			fmt.Println("s = ", s)

			if !ecdsa.Verify(&key.PublicKey, msgHash[:], r, s) {
				fmt.Println("couldn't verify signature: ") // + val.String())
				fmt.Println("msgHash = ", msgHash)
				fmt.Println("r = ", r)
				fmt.Println("s = ", s)
			}

			registerState.SignatureR = r.Bytes()
			registerState.SignatureS = s.Bytes()
			// fmt.Println("writing: ", registerState.String())
			ack, err := conf.Write(&registerState)
			if err != nil {
				dief("error writing: %v", err)
			}
			// fmt.Println("w " + ack.Reply.String())
			if ack.Reply.Timestamp > registerState.C.Timestamp {
				registerState.C.Timestamp = ack.Reply.Timestamp
			}
			time.Sleep(100 * time.Second)
		} else {
			// Reader client
			val, err := conf.Read(&byzq.Key{Key: content.Key})
			// TODO add Byzantine behavior by changing return value and detect verify failure.
			if err != nil {
				dief("error reading: %v", err)
			}
			if val.Reply.C.Timestamp > registerState.C.Timestamp {
				//TODO this should not happen if signature verification fails
				registerState.C.Timestamp = val.Reply.C.Timestamp
			}
			msg, err := val.Reply.C.Marshal()
			if err != nil {
				dief("failed to marshal msg for verify: %v", err)
			}
			fmt.Println("content = ", val.Reply.C.String())
			fmt.Println("msg = ", msg)

			msgHash := sha256.Sum256(msg)
			r := new(big.Int).SetBytes(val.Reply.SignatureR)
			s := new(big.Int).SetBytes(val.Reply.SignatureS)
			// s.Add(s, one) // Byzantine behavior (add 1 to signature field)

			if !ecdsa.Verify(&key.PublicKey, msgHash[:], r, s) {
				fmt.Println("couldn't verify signature: ") // + val.String())
				fmt.Println("msgHash = ", msgHash)
				fmt.Println("r = ", r)
				fmt.Println("s = ", s)
			}
			registerState = *val.Reply
			// fmt.Println("read: " + registerState.String())
			time.Sleep(10000 * time.Millisecond)
		}
	}
}

func dief(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprint(os.Stderr, "\n")
	flag.Usage()
	os.Exit(2)
}
