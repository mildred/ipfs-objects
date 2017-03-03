package net

import (
	"context"
	"io"
	"ipobj"
	"log"

	cid "github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

var _ ipobj.Network = &Network{}

func (net *Network) Id() []byte {
	return []byte(net.id)
}

func (net *Network) Providers(ctx context.Context, obj ipobj.ObjAddr) (<-chan *ipobj.PeerInfo, error) {
	contentid, err := cid.Cast(obj)
	if err != nil {
		return nil, err
	}
	size := 4
	res := make(chan *ipobj.PeerInfo, 0)
	respond := func(info *ipobj.PeerInfo) (cont bool) {
		cont = false
		defer recover()
		res <- info
		cont = true
		return
	}
	go func() {
		for {
			ctx2, cancel := context.WithCancel(ctx)
			c := net.client.FindProvidersAsync(ctx2, contentid, size)
			for i := 0; i < size; i++ {
				var peer pstore.PeerInfo
				peer = <-c
				var peers []ipobj.PeerInfo
				if len(peer.ID) > 0 {
					peers = append(peers, decodePeerInfo(peer))
				}
				for _, p := range peers {
					log.Printf("peer: %#v", p)
					if !respond(&p) {
						cancel()
						return
					}
				}
			}
			size = size * 2
		}
	}()
	return res, nil
}

func (net *Network) GetObject(ctx context.Context, obj ipobj.ObjAddr) (io.Reader, error) {
	id, err := cid.Cast(obj)
	if err != nil {
		return nil, err
	}
	block, err := net.exchange.GetBlock(ctx, id)
	if err != nil {
		return nil, err
	}

	return bytesToReader(block.RawData()), nil
}

func (net *Network) GetRecord(ctx context.Context, record string) <-chan *ipobj.Record {
	records := net.client.GetValuesAsync(ctx, record, -1)
	resChan := make(chan *ipobj.Record, 0)
	go func() {
		for {
			rec := <-records
			if rec == nil {
				close(resChan)
				break
			} else {
				resChan <- &ipobj.Record{
					PeerId:  []byte(rec.From),
					Content: rec.Val,
				}
			}
		}
	}()
	return resChan
}

func (net *Network) ProvideObject(ctx context.Context, obj ipobj.ObjAddr, provide bool) error {
	id, err := cid.Cast(obj)
	if err != nil {
		return err
	}
	if provide {
		net.store.list[string(id.Bytes())] = true
		return net.client.Provide(ctx, id)
	} else {
		delete(net.store.list, string(id.Bytes()))
		return nil // TODO: remove the block from the DHT
	}
}

func (net *Network) ProvideRecord(ctx context.Context, key string, rec []byte) error {
	return net.client.PutValue(ctx, key, rec)
}

func (net *Network) GetRecordFrom(ctx context.Context, peerId []byte, key string) ([]byte, error) {
	rec, err := net.client.GetValueFromPeer(ctx, peer.ID(peerId), key, false)
	if err != nil {
		return nil, err
	}
	return rec.Val, nil
}

func (net *Network) UpdatePeerRecord(ctx context.Context, peerId []byte, key string, record []byte) error {
	return net.client.PutValueToPeer(ctx, peer.ID(peerId), key, record)
}
