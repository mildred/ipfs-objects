package main

import (
	"context"
	"flag"
	"log"

	ipfs_objects "ipfs-objects"
	"ipobj"

	ic "gx/ipfs/QmfWDLQjGjVe4fr5CoztYW2DYYjRysMJrFe1RCsXLPTf46/go-libp2p-crypto"
)

func Main() {
	flag.Parse()

	var err error
	var config ipfs_object.NetworkConfig
	var net ipobj.Network

	net, err = ipfs_objects.NewNetwork(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}

	_ = net
}
