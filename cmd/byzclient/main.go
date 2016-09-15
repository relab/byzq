package main

import (
	"crypto/ecdsa"
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

	keyFile := "my-key.pem"

	// Crypto (TODO clean up later)
	// See https://golang.org/src/crypto/tls/generate_cert.go
	key := new(ecdsa.PrivateKey)

	f, err := os.Open(keyFile)
	if err != nil {
		dief("key file not found: %v", err)
	} else {
		b := make([]byte, 500)
		_, err := f.Read(b)
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

	ids := mgr.NodeIDs()

	// var qspec byzq.QuorumSpec
	var qspec *AuthDataQ
	switch *protocol {
	case "byzq":
		// 	qspec, err = NewByzQ(len(ids))
		// case "authq":
		qspec, err = NewAuthDataQ(len(ids), key, &key.PublicKey)
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

	registerState := &byzq.Value{C: content}
	for {
		if *writer {
			// Writer client
			k := rand.Intn(1 << 8)
			registerState.C.Value = strconv.Itoa(k)
			registerState.C.Timestamp++

			// if p, ok := qspec.(byzq.PreFn); ok {
			// 	err := p.PreWrite(*registerState)
			// 	if err != nil {
			// 		dief("failed to sign message: %v", err)
			// 	}
			// }
			registerState, err = qspec.Sign(registerState.C)
			if err != nil {
				dief("failed to sign message: %v", err)
			}
			ack, err := conf.Write(registerState)
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
			if err != nil {
				dief("error reading: %v", err)
			}
			if val.Reply.C.Timestamp > registerState.C.Timestamp {
				registerState.C.Timestamp = val.Reply.C.Timestamp
			}
			registerState = val.Reply
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
