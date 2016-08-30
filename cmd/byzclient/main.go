package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/relab/byzq/proto/byzq"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const localAddrs = ":8080,:8081,:8082,:8083,:8084"

func main() {
	saddrs := flag.String("addrs", localAddrs, "server addresses separated by ','")
	writer := flag.Bool("writer", false, "set this client to be writer only (default is reader only)")

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
			grpc.WithTimeout(1000*time.Millisecond),
			grpc.WithTransportCredentials(clientCreds),
		),
	)
	if err != nil {
		dief("error creating manager: %v", err)
	}
	defer mgr.Close()

	ids := mgr.NodeIDs()
	byzQSpec, err := NewByzQ(len(ids))
	if err != nil {
		dief("%v", err)
	}
	conf, err := mgr.NewConfiguration(ids, byzQSpec, time.Second)
	if err != nil {
		dief("error creating config: %v", err)
	}

	registerState := byzq.State{Value: int64(rand.Intn(1 << 8))}
	for {
		if *writer {
			// Writer client
			registerState.Value = int64(rand.Intn(1 << 8))
			// registerState.Timestamp++
			fmt.Println("writing: ", registerState.String())
			ack, err := conf.Write(&registerState)
			if err != nil {
				dief("error writing: %v", err)
			}
			fmt.Println("w " + ack.Reply.String())
			if ack.Reply.Timestamp > registerState.Timestamp {
				registerState.Timestamp = ack.Reply.Timestamp
			}
			time.Sleep(1 * time.Second)
		} else {
			// Reader client
			val, err := conf.Read(&byzq.Empty{})
			if err != nil {
				dief("error reading: %v", err)
			}
			if val.Reply.Timestamp > registerState.Timestamp {
				registerState.Timestamp = val.Reply.Timestamp
			}
			fmt.Println("read: " + registerState.String())
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func dief(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprint(os.Stderr, "\n")
	flag.Usage()
	os.Exit(2)
}
