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
	ma "github.com/multiformats/go-multiaddr"
)

func resolve(cfg Config, args []string) error {
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
	config.ListenAddresses, err = cfg.ListenAddrs.Get()
	if err != nil {
		return err
	}

	net, err := ipfs_objects.NewNetwork(context.Background(), config, ipobj.NullPeer, sk)
	if err != nil {
		return err
	}

	fmt.Printf("Peer id: %s\n", base58.Encode(net.Id()))
	// list out our addresses
	addrs, err := net.InterfaceListenAddresses()
	if err != nil {
		return err
	}
	fmt.Printf("Swarm listening at:\n")
	for _, a := range addrs {
		fmt.Printf("  - %s\n", a)
	}

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
				var p *ipobj.PeerInfo
				select {
				case <-ctx.Done():
					break loop
				case p = <-peers:
					if p == nil {
						fmt.Printf("%s: no provider\n", record)
						break loop
					}
					break
				}

				fmt.Printf("%s: possible provider: %s\n", record, base58.Encode(p.Id))
				for _, a := range p.Addrs {
					fmt.Printf("  - %v\n", ma.Cast(a))
				}
				data, err := net.GetRecordFrom(ctx, p.Id, record)
				if err != nil {
					fmt.Printf("%s: error from %s: %v\n", record, base58.Encode(p.Id), err)
					continue
				}
				fmt.Printf("%s: response from: %v\n\t%v\n", record, base58.Encode(p.Id), data)
			}
		}(record)
	}

	wg.Wait()
	return nil
}
