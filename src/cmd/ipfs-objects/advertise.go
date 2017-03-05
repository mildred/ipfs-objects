package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"ipobj"
	ipnet "ipobj-net"
	osr "ipobj-osr"

	base58 "github.com/jbenet/go-base58"
	ic "github.com/libp2p/go-libp2p-crypto"
)

type advertisePeer struct {
	ipobj.NullPeerType
	values map[string][]byte
}

func (ap *advertisePeer) GetRecord(key string) ([]byte, error) {
	fmt.Printf("GetRecord %s\n", key)
	return ap.values[key], nil
}

func (ap *advertisePeer) NewRecord(key string, value []byte, peer []byte) {
	newRec, err := osr.Decode(value)
	if err != nil {
		fmt.Printf("%s: new record from %s: decode error %s", key, base58.Encode(peer), err)
		return
	}

	newKey, err := newRec.Path()
	if err != nil {
		fmt.Printf("%s: new record from %s: path error %s", key, base58.Encode(peer), err)
		return
	}
	if "/iprs"+newKey != key {
		fmt.Printf("%s: new record from %s: mismatching path /iprs%s", key, base58.Encode(peer), newKey)
		return
	}

	recData, hasRec := ap.values[key]
	if !hasRec {
		fmt.Printf("%s: new record from %s\n", key, base58.Encode(peer))
		return
	}

	rec, err := osr.Decode(recData)
	if err != nil {
		fmt.Printf("%s: decode OSR error: %s", key, err)
		return
	}

	if newRec.Order == rec.Order {
		fmt.Printf("%s: same record from %s\n", key, base58.Encode(peer))
	} else if newRec.Order < rec.Order {
		fmt.Printf("%s: old record from %s (%d)\n", key, base58.Encode(peer), newRec.Order)
	} else {
		fmt.Printf("%s: newer record from %s (%d)\n", key, base58.Encode(peer), newRec.Order)
		ap.values[key] = value
	}
}

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

	recordFile := f.Arg(0)
	recordKey := f.Arg(1)

	recordData, err := ioutil.ReadFile(recordFile)
	if err != nil {
		return err
	}

	if recordKey == "" {
		rec, err := osr.Decode(recordData)
		if err != nil {
			return err
		}
		recordKey, err = rec.Path()
		if err != nil {
			return err
		}
		recordKey = "/iprs" + recordKey
	}

	var peer *advertisePeer = new(advertisePeer)
	peer.values = map[string][]byte{
		recordKey: recordData,
	}

	var config ipnet.NetworkConfig
	config.ListenAddresses, err = cfg.ListenAddrs.Get()
	if err != nil {
		return err
	}
	net, err := ipnet.NewNetwork(context.Background(), config, peer, sk)
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

	fmt.Printf("Advertise: %s\n", recordKey)

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
