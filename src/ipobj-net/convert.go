package net

import (
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"ipobj"
)

func encodePeerInfo(i ipobj.PeerInfo) (res pstore.PeerInfo) {
	res.ID = peer.ID(i.Id)
	for _, a := range i.Addrs {
		multiaddr, err := ma.NewMultiaddrBytes([]byte(a))
		if err != nil {
			panic(err)
		}
		res.Addrs = append(res.Addrs, multiaddr)
	}
	return
}

func decodePeerInfo(i pstore.PeerInfo) (res ipobj.PeerInfo) {
	res.Id = []byte(i.ID)
	for _, a := range i.Addrs {
		res.Addrs = append(res.Addrs, a.Bytes())
	}
	return
}
