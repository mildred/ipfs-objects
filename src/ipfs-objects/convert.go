package ipfs_objects

import (
	ma "gx/ipfs/QmUAQaWbKxGCUTuoQVvvicbQNZ9APF5pDGWyAZSe93AtKH/go-multiaddr"
	pstore "gx/ipfs/QmeXj9VAjmYQZxpmVz7VzccbJrpmr8qkCDSjfVNsPTWTYU/go-libp2p-peerstore"
	"gx/ipfs/QmfMmLGoKzCHDN7cGgk64PJr4iipzidDRME8HABSJqvmhC/go-libp2p-peer"

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
	res.Id = ipobj.PeerId(i.ID)
	for _, a := range i.Addrs {
		res.Addrs = append(res.Addrs, a.Bytes())
	}
	return
}
