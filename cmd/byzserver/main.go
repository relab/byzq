package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/relab/byzq/proto/byzq"
)

type register struct {
	sync.RWMutex
	state byzq.State
}

func main() {
	port := flag.String("port", "8080", "port to listen on")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatal(err)
	}

	register := &register{}

	grpcServer := grpc.NewServer()
	byzq.RegisterRegisterServer(grpcServer, register)
	log.Fatal(grpcServer.Serve(l))
}

func (r *register) Read(ctx context.Context, e *byzq.Empty) (*byzq.State, error) {
	r.RLock()
	state := r.state
	r.RUnlock()
	return &state, nil
}

func (r *register) Write(ctx context.Context, s *byzq.State) (*byzq.WriteResponse, error) {
	wr := &byzq.WriteResponse{}
	r.Lock()
	if s.Timestamp > r.state.Timestamp {
		r.state = *s
		wr.Timestamp = s.Timestamp
		wr.Written = true
	}
	r.Unlock()
	return wr, nil
}
