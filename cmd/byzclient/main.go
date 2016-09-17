package main

import (
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

	key := readKeyfile()
	// var qspec byzq.QuorumSpec
	var qspec *byzq.AuthDataQ
	switch *protocol {
	// case "byzq":
	// 	qspec, err = NewByzQ(len(ids))
	case "authq":
		qspec, err = byzq.NewAuthDataQ(len(ids), key, &key.PublicKey)
	}
	if err != nil {
		dief("%v", err)
	}
	conf, err := mgr.NewConfiguration(ids, qspec, time.Second)
	if err != nil {
		dief("error creating config: %v", err)
	}

	registerState := &byzq.Content{
		Key:       "Hein",
		Value:     "Meling",
		Timestamp: -1,
	}

	for {
		if *writer {
			// Writer client
			registerState.Value = strconv.Itoa(rand.Intn(1 << 8))
			registerState.Timestamp = qspec.IncWTS()
			signedState, err := qspec.Sign(registerState)
			if err != nil {
				dief("failed to sign message: %v", err)
			}
			ack, err := conf.Write(signedState)
			if err != nil {
				dief("error writing: %v", err)
			}
			fmt.Println("WriteReturn " + ack.Reply.String())
			time.Sleep(100 * time.Second)
		} else {
			// Reader client
			val, err := conf.Read(&byzq.Key{Key: registerState.Key})
			if err != nil {
				dief("error reading: %v", err)
			}
			registerState = val.Reply.C
			fmt.Println("ReadReturn: " + registerState.String())
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
