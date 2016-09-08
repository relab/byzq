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

	"github.com/relab/byzq"
)

type register struct {
	sync.RWMutex
	state map[string]byzq.Value
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
	smap := make(map[string]byzq.Value)
	byzq.RegisterRegisterServer(grpcServer, &register{state: smap})
	log.Fatal(grpcServer.Serve(l))
}

func (r *register) Read(ctx context.Context, k *byzq.Key) (*byzq.Value, error) {
	r.RLock()
	value := r.state[k.Key]
	r.RUnlock()
	return &value, nil
}

func (r *register) Write(ctx context.Context, v *byzq.Value) (*byzq.WriteResponse, error) {
	wr := &byzq.WriteResponse{Timestamp: v.C.Timestamp}
	r.Lock()
	val, found := r.state[v.C.Key]
	if !found || v.C.Timestamp > val.C.Timestamp {
		r.state[v.C.Key] = *v
	}
	r.Unlock()
	return wr, nil
}
