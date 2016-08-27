package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/relab/byzq/proto/byzq"

	"google.golang.org/grpc"
)

const localAddrs = ":8080,:8081,:8082,:8083,:8084"

func main() {
	saddrs := flag.String("addrs", localAddrs, "server addresses separated by ','")

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

	mgr, err := byzq.NewManager(
		addrs,
		byzq.WithGrpcDialOptions(
			grpc.WithBlock(),
			grpc.WithInsecure(),
			grpc.WithTimeout(5*time.Second),
		),
	)
	if err != nil {
		dief("error creating manager: %v", err)
	}

	ids := mgr.NodeIDs()
	byzQSpec := NewByzQ(len(ids), 1) // todo(meling) change signature to produce error if wrong ids size.
	conf, err := mgr.NewConfiguration(ids, byzQSpec, time.Second)
	if err != nil {
		dief("error creating config: %v", err)
	}

	ack, err := conf.Write(&byzq.State{Timestamp: 9, Value: 42})
	if err != nil {
		dief("error writing: %v", err)
	}
	fmt.Println("w " + ack.Reply.String())

	val, err := conf.Read(&byzq.Empty{})
	if err != nil {
		dief("error reading: %v", err)
	}
	fmt.Println("r " + val.Reply.String())

	// _ = conf
}

func dief(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprint(os.Stderr, "\n")
	flag.Usage()
	os.Exit(2)
}
