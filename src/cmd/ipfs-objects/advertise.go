package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	ipfs_objects "ipfs-objects"
	"ipobj"

	base58 "github.com/jbenet/go-base58"
	ic "github.com/libp2p/go-libp2p-crypto"
)

type advertisePeer struct {
	ipobj.NullPeerType
	values map[string][]byte
}

func (ap *advertisePeer) GetRecord(key string) ([]byte, error) {
	return ap.values[key], nil
}

func (ap *advertisePeer) NewRecord(key string, value []byte, peer []byte) {}

func advertise(cfg Config, args []string) error {
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

	recordKey := f.Arg(0)
	recordData := []byte(f.Arg(1))

	var peer *advertisePeer = new(advertisePeer)
	peer.values = map[string][]byte{
		recordKey: recordData,
	}

	var config ipfs_objects.NetworkConfig
	config.ListenAddresses, err = cfg.ListenAddrs.Get()
	if err != nil {
		return err
	}
	net, err := ipfs_objects.NewNetwork(context.Background(), config, peer, sk)
	if err != nil {
		return err
	}

	fmt.Printf("Peer id: %s\n", base58.Encode(net.Id()))

	ctx := contextWithSignal(context.Background())

	for {
		deadline := time.Now().Add(interval)

		cid := ipobj.NewRecordObjAddr(recordKey)
		fmt.Printf("Advertise CID: %s\n", base58.Encode(cid))
		net.ProvideObject(ctx, cid, true)
		if err != nil {
			return err
		}

		// Sleep until next deadline
		ctx2, _ := context.WithDeadline(ctx, deadline)
		<-ctx2.Done()
	}
}
