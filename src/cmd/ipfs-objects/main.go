package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	ipfs_objects "ipfs-objects"
	"ipobj"

	ic "gx/ipfs/QmNiCwBNA8MWDADTFVq1BonUEJbS2SvjAoNkZZrhEwcuUi/go-libp2p-crypto"
)

func main() {
	var err error
	flag.Parse()

	switch flag.Arg(0) {
	case "keygen":
		err = keygen(flag.Args())
		break
	default:
		err = fmt.Errorf("Please specify a valid command")
		fallthrough
	case "help":
		fmt.Println("Available commands:")
		fmt.Println("\thelp")
		fmt.Println("\tkeygen")
		fmt.Println("Unavailable commands:")
		fmt.Println("\tadvertise - advertise naming record to root block")
		fmt.Println("\tresolve   - resolve naming record to root block")
		break
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

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

func daemon(args []string) error {
	var f flag.FlagSet
	var keyfile string
	f.StringVar(&keyfile, "k", "", "Secret key file")
	f.Parse(args[1:])

	if keyfile == "" {
		return fmt.Errorf("Please specify a key file with -k")
	}

	bytes, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return err
	}

	sk, err := ic.UnmarshalPrivateKey(bytes)
	if err != nil {
		return err
	}

	var config ipfs_objects.NetworkConfig
	var net ipobj.Network
	var client ipobj.Peer

	net, err = ipfs_objects.NewNetwork(context.Background(), config, client, sk)
	if err != nil {
		return err
	}

	_ = net
	return nil
}
