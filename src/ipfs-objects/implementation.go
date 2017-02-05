package ipfs_objects

import (
	"context"
	"io"
	"ipobj"

	"github.com/ipfs/go-ipfs/routing/supernode"
	mh "github.com/multiformats/go-multihash"
	cid "gx/ipfs/QmcTcsTvfaeEBRFo1TkFgT8sRmgi1n1LTZpecfVP8fzpGD/go-cid"
	pstore "gx/ipfs/QmeXj9VAjmYQZxpmVz7VzccbJrpmr8qkCDSjfVNsPTWTYU/go-libp2p-peerstore"
)

var _ ipobj.Network = &Network{}

type Network struct {
	client *supernode.Client
}

func NewNetwork() *Network {
	return &Network{}
}

func (net *Network) Providers(obj ObjAddr, updated bool) <-chan []PeerInfo {
	ctx, cancel := context.WithCancel(context.Background())
	contentid := cid.Cast(obj)
	size := 4
	res := make(chan []PeerInfo, 0)
	respond = func(info []PeerInfo) (cont bool) {
		cont = false
		defer recover()
		res <- info
		cont = true
		return
	}
	go func() {
		for {
			c := net.client.FindProvidersAsync(ctx, contentid, MaxProviders, size)
			for var i := 0; i < size; i++ {
				var peer pstore.PeerInfo
				peer <- c
				respond([]PeerInfo{peer})
			}
			size = size * 2
		}
	}()
	return res
}

func (net *Network) GetObject(ipobj.PeerAddr, ipobj.ObjAddr) io.Reader {
}

func (net *Network) Provide(obj ObjAddr, update bool) {
}

func (net *Network) Update(peer PeerAddr, obj ObjAddr) {
}
