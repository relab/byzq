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
	"google.golang.org/grpc/credentials"

	"github.com/relab/byzq/proto/byzq"
)

type register struct {
	sync.RWMutex
	state byzq.State
}

func main() {
	port := flag.String("port", "8080", "port to listen on")
	key := flag.String("key", "", "name of public/private key files (must share same prefix)")

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

	if *key == "" {
		log.Fatalln("required server keys not provided")
	}
	creds, err := credentials.NewServerTLSFromFile(*key+".pem", *key+".key")
	if err != nil {
		log.Fatalf("failed to load credentials %v", err)
	}
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	grpcServer := grpc.NewServer(opts...)
	byzq.RegisterRegisterServer(grpcServer, &register{})
	log.Fatal(grpcServer.Serve(l))
}

func (r *register) Read(ctx context.Context, e *byzq.Empty) (*byzq.State, error) {
	r.RLock()
	state := r.state
	r.RUnlock()
	return &state, nil
}

func (r *register) Write(ctx context.Context, s *byzq.State) (*byzq.WriteResponse, error) {
	wr := &byzq.WriteResponse{Timestamp: s.Timestamp}
	r.Lock()
	if s.Timestamp > r.state.Timestamp {
		r.state = *s
	}
	r.Unlock()
	return wr, nil
}
