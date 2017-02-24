package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	ipfs_objects "ipfs-objects"
	"ipobj"

	ic "github.com/libp2p/go-libp2p-crypto"
)

func advertise(args []string) error {
	var f flag.FlagSet
	var keyfile string
	var interval time.Duration
	f.StringVar(&keyfile, "k", "", "Secret key file")
	f.DurationVar(&interval, "t", time.Hour, "Time interval between advertisements")
	f.Parse(args[1:])

	var err error
	var sk ic.PrivKey
	if keyfile == "" {
		sk, err = dummySecretKey()
	} else {
		sk, err = readKeyFile(keyfile)
	}

	var config ipfs_objects.NetworkConfig
	net, err := ipfs_objects.NewNetwork(context.Background(), config, ipobj.NullPeer, sk)
	if err != nil {
		return err
	}

	ctx, stop := context.WithCancel(context.Background())
	c := make(chan os.Signal, 5)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for {
			sig := <-c
			fmt.Fprintf(os.Stderr, "Received signal %v: stop operations", sig)
			stop()
		}
	}()

	record := f.Arg(0)
	recordData := []byte(f.Arg(1))

	for {
		deadline := time.Now().Add(interval)

		err := net.ProvideRecord(ctx, record, recordData)
		if err != nil {
			return err
		}

		// Sleep until next deadline
		ctx2, _ := context.WithDeadline(ctx, deadline)
		<-ctx2.Done()
	}
}
