package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
)

const keyFile = "my-key.pem"

var curve = elliptic.P256()

func genKeyfile() {
	f, err := os.OpenFile("ec-key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		dief("failed to open key file for writing: %v", err)
	}
	defer f.Close()

	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		dief("failed to generate keys: %v", err)
	}

	ec, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		dief("failed to marshal key: %v\n %v", err, key)
	}
	err = pem.Encode(f, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: ec,
	})
	if err != nil {
		dief("failed to PEM encode key: %v\n %v", err, key)
	}
}

func readKeyfile() *ecdsa.PrivateKey {
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
	return key
}
