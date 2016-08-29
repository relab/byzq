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
	byzQSpec, err := NewByzQ(len(ids))
	if err != nil {
		dief("%v", err)
	}
	conf, err := mgr.NewConfiguration(ids, byzQSpec, time.Second)
	if err != nil {
		dief("error creating config: %v", err)
	}

	for {
		state := byzQSpec.newWrite(int64(rand.Intn(1 << 8)))
		fmt.Println("writing:", state)
		ack, err := conf.Write(state)
		if err != nil {
			dief("error writing: %v", err)
		}
		fmt.Println("w " + ack.Reply.String())

		time.Sleep(1 * time.Second)

		val, err := conf.Read(&byzq.Empty{})
		if err != nil {
			dief("error reading: %v", err)
		}
		if val.Reply.Timestamp > byzQSpec.wts {

		}
		fmt.Println("r " + val.Reply.String())
		time.Sleep(1 * time.Second)
	}
}

func dief(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprint(os.Stderr, "\n")
	flag.Usage()
	os.Exit(2)
}
