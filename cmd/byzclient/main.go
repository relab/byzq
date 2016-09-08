package main

import (
	"flag"
	"fmt"
	"log"
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

	// certFile := "cert/ca.pem"
	// serverName := "x.test.youtube.com"
	// pemCerts, err := ioutil.ReadFile(certFile)
	// if err != nil {
	// 	dief("error creating credentials: %v", err)
	// }

	// for len(pemCerts) > 0 {
	// 	var block *pem.Block
	// 	block, pemCerts = pem.Decode(pemCerts)
	// 	if block == nil {
	// 		break
	// 	}
	// 	if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
	// 		continue
	// 	}

	// 	cert, err := x509.ParseCertificate(block.Bytes)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	//  .CheckSignature()
	// 	// cert.Signature
	// 	fmt.Printf("%v: %v\n", cert.SignatureAlgorithm, cert.Signature)
	// 	// s.AddCert(cert)
	// 	// ok = true
	// }

	// // cert, err := x509.ParseCertificate(b)
	// // if err != nil {
	// // 	dief("credentials: failed to append certificatesxxx")
	// // }
	// cp := x509.NewCertPool()
	// if !cp.AppendCertsFromPEM(pemCerts) {
	// 	dief("credentials: failed to append certificates")
	// }
	// clientCreds := credentials.NewTLS(&tls.Config{ServerName: serverName, RootCAs: cp})

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

	// TODO(meling) Sign this content object (for the authq protocol)

	registerState := byzq.Value{C: content}
	for {
		if *writer {
			// Writer client
			k := rand.Intn(1 << 8)
			registerState.C.Value = strconv.Itoa(k)
			registerState.C.Timestamp++
			fmt.Println("writing: ", registerState.String())
			ack, err := conf.Write(&registerState)
			if err != nil {
				dief("error writing: %v", err)
			}
			fmt.Println("w " + ack.Reply.String())
			if ack.Reply.Timestamp > registerState.C.Timestamp {
				registerState.C.Timestamp = ack.Reply.Timestamp
			}
			time.Sleep(1 * time.Second)
		} else {
			// Reader client
			val, err := conf.Read(&byzq.Key{Key: content.Key})
			if err != nil {
				dief("error reading: %v", err)
			}
			if val.Reply.C.Timestamp > registerState.C.Timestamp {
				registerState.C.Timestamp = val.Reply.C.Timestamp
			}
			registerState = *val.Reply
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
