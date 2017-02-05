package main

import (
	"flag"
	ipfs_objects "ipfs-objects"
	"ipobj"
)

func Main() {
	flag.Parse()

	var net ipobj.Network = ipfs_objects.NewNetwork()
	_ = net
}
