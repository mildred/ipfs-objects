package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"

	ipfs_objects "ipfs-objects"
	"ipobj"

	base58 "github.com/jbenet/go-base58"
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

	fmt.Printf("Peer id: %s\n", base58.Encode(net.Id()))

	ctx := contextWithSignal(context.Background())
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

			cid := ipobj.NewRecordObjAddr(record)
			fmt.Printf("Request CID: %s\n", base58.Encode(cid))
			peers, err := net.Providers(ctx2, cid)
			if err != nil {
				fmt.Printf("%s: error: %v\n", record, err)
				return
			}

		loop:
			for {
				var pl []ipobj.PeerInfo
				select {
				case <-ctx.Done():
					break loop
				case pl = <-peers:
					if pl == nil {
						fmt.Printf("%s: no provider\n", record)
						break loop
					}
					break
				}

				for _, p := range pl {
					fmt.Printf("%s: possible provider: %s %#v\n", record, base58.Encode(p.Id), p)
					data, err := net.GetRecordFrom(ctx, p.Id, record)
					if err != nil {
						fmt.Printf("%s: error from %s: %v\n", record, base58.Encode(p.Id), err)
						continue
					}
					fmt.Printf("%s: response from: %v\n\t%v\n", record, base58.Encode(p.Id), data)
				}
			}
		}(record)
	}

	wg.Wait()
	return nil
}
