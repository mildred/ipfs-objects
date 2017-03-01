package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	ipfs_objects "ipfs-objects"
	"ipobj"

	"github.com/ipfs/go-log"
	ic "github.com/libp2p/go-libp2p-crypto"
)

func main() {
	var f flag.FlagSet
	var debug bool
	var cfg Config
	cfg.Flags(&f)
	f.BoolVar(&debug, "debug", false, "debug logging")
	f.Parse(os.Args[1:])

	var err error

	if debug {
		log.SetDebugLogging()
	}

	switch f.Arg(0) {
	case "keygen":
		err = keygen(f.Args())
		break
	case "resolve":
		err = resolve(cfg, f.Args())
		break
	case "advertise":
		err = advertise(cfg, f.Args())
		break
	default:
		err = fmt.Errorf("Please specify a valid command: %s invalid", f.Arg(0))
		fallthrough
	case "help":
		fmt.Println("Available commands:")
		fmt.Println("\thelp      - this help")
		fmt.Println("\tkeygen    - generate secret key")
		fmt.Println("\tresolve   - resolve naming record to root block")
		fmt.Println("\tadvertise - advertise naming record to root block")
		break
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func readKeyFile(keyfile string) (ic.PrivKey, error) {
	bytes, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}

	sk, err := ic.UnmarshalPrivateKey(bytes)
	if err != nil {
		return nil, err
	}
	return sk, err
}

func dummySecretKey() (ic.PrivKey, error) {
	sk, _, err := ic.GenerateEd25519Key(rand.Reader)
	return sk, err
}

func daemon(args []string) error {
	var f flag.FlagSet
	var keyfile string
	f.StringVar(&keyfile, "k", "", "Secret key file")
	f.Parse(args[1:])

	if keyfile == "" {
		return fmt.Errorf("Please specify a key file with -k")
	}

	sk, err := readKeyFile(keyfile)

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

func contextWithSignal(ctx context.Context) context.Context {
	return ctx
}

func contextWithSignalBug(ctx context.Context) context.Context {
	ctx2, stop := context.WithCancel(ctx)
	c := make(chan os.Signal, 5)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			sig := <-c
			fmt.Fprintf(os.Stderr, "Received signal %v: stop operations", sig)
			stop()
		}
	}()
	return ctx2
}
