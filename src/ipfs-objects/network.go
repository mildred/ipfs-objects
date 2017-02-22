package ipfs_objects

import (
	"context"
	"io"
	"ipobj"

	cid "gx/ipfs/QmcTcsTvfaeEBRFo1TkFgT8sRmgi1n1LTZpecfVP8fzpGD/go-cid"
	pstore "gx/ipfs/QmeXj9VAjmYQZxpmVz7VzccbJrpmr8qkCDSjfVNsPTWTYU/go-libp2p-peerstore"
)

var _ ipobj.Network = &Network{}

func (net *Network) Providers(ctx context.Context, obj ipobj.ObjAddr, updated bool) (<-chan []ipobj.PeerInfo, error) {
	ctx, cancel := context.WithCancel(ctx)
	contentid, err := cid.Cast(obj)
	if err != nil {
		return nil, err
	}
	size := 4
	res := make(chan []ipobj.PeerInfo, 0)
	respond := func(info []ipobj.PeerInfo) (cont bool) {
		cont = false
		defer recover()
		res <- info
		cont = true
		return
	}
	go func() {
		for {
			c := net.client.FindProvidersAsync(ctx, contentid, size)
			for i := 0; i < size; i++ {
				var peer pstore.PeerInfo
				peer = <-c
				respond([]ipobj.PeerInfo{decodePeerInfo(peer)})
			}
			size = size * 2
		}
	}()
	return res, nil
}

func (net *Network) GetObject(ipobj.PeerAddr, ipobj.ObjAddr) io.Reader {
}

func (net *Network) Provide(obj ipobj.ObjAddr, update bool) {
}

func (net *Network) Update(peer ipobj.PeerAddr, obj ipobj.ObjAddr) {
}
