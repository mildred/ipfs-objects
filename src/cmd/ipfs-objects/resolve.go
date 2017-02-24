package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"

	ipfs_objects "ipfs-objects"
	"ipobj"

	"github.com/jbenet/go-base58"

	ic "github.com/libp2p/go-libp2p-crypto"
)

func resolve(args []string) error {
	var f flag.FlagSet
	var keyfile string
	var timeout time.Duration
	f.StringVar(&keyfile, "k", "", "Secret key file")
	f.DurationVar(&timeout, "t", 0, "Timeout")
	f.Parse(args[1:])

	var err error
	var sk ic.PrivKey
	if keyfile == "" {
		sk, err = dummySecretKey()
	} else {
		sk, err = readKeyFile(keyfile)
	}

	config := ipfs_objects.NetworkConfig{
		ClientOnly: true,
	}

	net, err := ipfs_objects.NewNetwork(context.Background(), config, ipobj.NullPeer, sk)
	if err != nil {
		return err
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	for _, record := range f.Args() {
		wg.Add(1)
		go func(record string) {
			defer wg.Done()
			var ctx2 context.Context
			var cancel context.CancelFunc

			if timeout != 0 {
				ctx2, cancel = context.WithTimeout(ctx, timeout)
			} else {
				ctx2, cancel = context.WithCancel(ctx)
			}
			defer cancel()

			records := net.GetRecord(ctx2, record)

			res := <-records
			if res == nil {
				fmt.Printf("%s: not found\n", record)
			} else {
				fmt.Printf("%s: response from %v\n", base58.Encode(res.PeerId))
			}
		}(record)
	}

	wg.Wait()
	return nil
}
