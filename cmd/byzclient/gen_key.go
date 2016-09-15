package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
)

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
