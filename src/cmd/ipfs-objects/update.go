package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"ipobj"
	ipnet "ipobj-net"
	osr "ipobj-osr"

	base58 "github.com/jbenet/go-base58"
	ic "github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
)

func update(cfg Config, args []string) error {
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

	config := ipnet.NetworkConfig{
		ClientOnly: true,
	}
	config.ListenAddresses, err = cfg.ListenAddrs.Get()
	if err != nil {
		return err
	}

	net, err := ipnet.NewNetwork(context.Background(), config, ipobj.NullPeer, sk)
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
	defer wg.Wait()

	for _, recordFile := range f.Args() {
		recordData, err := ioutil.ReadFile(recordFile)
		if err != nil {
			return err
		}
		rec, err := osr.Decode(recordData)
		if err != nil {
			return err
		}
		recordKey, err := rec.Path()
		if err != nil {
			return err
		}
		recordKey = "/iprs" + recordKey

		wg.Add(1)
		go func(recordKey string, recordData []byte, rec *osr.Record) {
			defer wg.Done()
			var ctx2 context.Context
			var cancel context.CancelFunc

			if timeout != 0 {
				ctx2, cancel = context.WithTimeout(ctx, timeout)
			} else {
				ctx2, cancel = context.WithCancel(ctx)
			}
			defer cancel()

			cid := ipobj.NewRecordObjAddr(recordKey)
			fmt.Printf("Record:     %s\nRecord CID: %s\n", recordKey, base58.Encode(cid))
			peers, err := net.Providers(ctx2, cid)
			if err != nil {
				fmt.Printf("%s: error: %v\n", recordKey, err)
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
						fmt.Printf("%s: no provider\n", recordKey)
						break loop
					}
					break
				}

				fmt.Printf("%s: possible provider: %s\n", recordKey, base58.Encode(p.Id))
				for _, a := range p.Addrs {
					fmt.Printf("  - %v\n", ma.Cast(a))
				}
				data, err := net.GetRecordFrom(ctx, p.Id, recordKey)
				if err != nil {
					fmt.Printf("%s: error from %s: %v\n", recordKey, base58.Encode(p.Id), err)
					continue
				}
				fmt.Printf("%s: response from: %v\n", recordKey, base58.Encode(p.Id))
				err = updateRecord(ctx, net, recordKey, recordData, rec, p.Id, data)
				if err != nil {
					fmt.Printf("%s: error from %s: %s\n", recordKey, base58.Encode(p.Id), err)
				}
			}
		}(recordKey, recordData, rec)
	}

	return nil
}

func updateRecord(ctx context.Context, net *ipnet.Network, key string, baseRecData []byte, baseRec *osr.Record, peerId []byte, newRecData []byte) error {
	newRec, err := osr.Decode(newRecData)
	if err != nil {
		return err
	}

	newKey, err := newRec.Path()
	if err != nil {
		return err
	}

	if "/iprs"+newKey != key {
		fmt.Errorf("Mismatching key: /iprs%s", newKey)
	}

	if newRec.Order == baseRec.Order {
		fmt.Printf("%s: same record from %s\n", key, base58.Encode(peerId))
		return nil
	} else if newRec.Order > baseRec.Order {
		fmt.Printf("%s: newer record from %s (%d)\n", key, base58.Encode(peerId), newRec.Order)
		return nil
	}
	fmt.Printf("%s: old record from %s (%d)\n", key, base58.Encode(peerId), newRec.Order)

	return net.UpdatePeerRecord(ctx, peerId, key, baseRecData)
}
