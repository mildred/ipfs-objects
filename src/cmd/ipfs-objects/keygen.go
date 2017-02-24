package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"

	ic "github.com/libp2p/go-libp2p-crypto"
)

func keygen(args []string) error {
	var f flag.FlagSet
	var out string
	var keytype string
	var keysize int
	f.StringVar(&out, "o", "", "Output file")
	f.StringVar(&keytype, "t", "ed25519", "Key Type")
	f.IntVar(&keysize, "s", 4096, "Key Size (for RSA)")
	f.Parse(args[1:])

	if out == "" {
		return fmt.Errorf("Please specify a filename with -o")
	}

	var sk ic.PrivKey
	var err error

	switch keytype {
	case "ed25519":
		sk, _, err = ic.GenerateEd25519Key(rand.Reader)
	case "rsa":
		sk, _, err = ic.GenerateKeyPairWithReader(ic.RSA, keysize, rand.Reader)
	default:
		err = fmt.Errorf("Supported key types: rsa, ed25519")
	}

	if err != nil {
		return err
	}

	bytes, err := sk.Bytes()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(out, bytes, 0600)
}
