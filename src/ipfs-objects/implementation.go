package ipfs_objects

import (
	"context"
	"fmt"
	"ipobj"
	"net"

	core "github.com/ipfs/go-ipfs/core"
	exchange "github.com/ipfs/go-ipfs/exchange"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"

	p2phost "gx/ipfs/QmPsRtodRuBUir32nz5v4zuSBTSszrR1d3fA6Ahb6eaejj/go-libp2p-host"
	dht "gx/ipfs/QmRG9fdibExi5DFy8kzyxF76jvZVUb2mQBUSMNP1YaYn9M/go-libp2p-kad-dht"
	ds "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore"
	mamask "gx/ipfs/QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv/multiaddr-filter"
	metrics "gx/ipfs/QmY2otvyPM2sTaDsczo7Yuosg98sUMCJ9qx1gpPaAPTS9B/go-libp2p-metrics"
	routing "gx/ipfs/QmbkGVaN9W6RYJK4Ws5FvMKXKDqdRQ5snhtaa92qP6L8eU/go-libp2p-routing"
	rhost "gx/ipfs/QmdzDdLZ7nj133QvNHypyS9Y39g35bMFk5DJ2pmX7YqtKU/go-libp2p/p2p/host/routed"
	pstore "gx/ipfs/QmeXj9VAjmYQZxpmVz7VzccbJrpmr8qkCDSjfVNsPTWTYU/go-libp2p-peerstore"
	smux "gx/ipfs/QmeZBgYBHvxMukGK5ojg28BCNLB9SeXqT7XXg6o7r2GbJy/go-stream-muxer"
	peer "gx/ipfs/QmfMmLGoKzCHDN7cGgk64PJr4iipzidDRME8HABSJqvmhC/go-libp2p-peer"
	ic "gx/ipfs/QmfWDLQjGjVe4fr5CoztYW2DYYjRysMJrFe1RCsXLPTf46/go-libp2p-crypto"
)

var _ ipobj.Network = &Network{}

type Network struct {

	// "github.com/ipfs/go-ipfs/routing/supernode"
	// or *supernode.Client
	client routing.IpfsRouting

	// See go-ipfs/core/builder.go
	exchange exchange.Interface

	// Network should implement blockstore to give it to bitswap
	// Network should contain bitswap service to get nodes as implementation to
	// exchange interface

	store *PeerBlockstore
}

func parseAddrs(addrs []string) ([]*net.IPNet, error) {
	var addrfilter []*net.IPNet
	for _, s := range addrs {
		f, err := mamask.NewMask(s)
		if err != nil {
			return nil, fmt.Errorf("incorrectly formatted address filter in config: %s", s)
		}
		addrfilter = append(addrfilter, f)
	}
	return addrfilter, nil
}

type NetworkConfig struct {
	ClientOnly    bool
	RudeBitswap   bool
	DialBlockList []string
}

func NewNetwork(ctx context.Context, config NetworkConfig, peerObj ipobj.Peer, secretKey ic.PrivKey) (*Network, error) {

	var err error

	// Get ID from crypto key
	var id peer.ID
	id, err = peer.IDFromPublicKey(secretKey.GetPublic())
	if err != nil {
		return nil, err
	}

	// Add ID to peer store
	var ps pstore.Peerstore = pstore.NewPeerstore()
	ps.AddPrivKey(id, secretKey)
	ps.AddPubKey(id, secretKey.GetPublic())

	// Parse address filter
	var fs []*net.IPNet
	fs, err = parseAddrs(config.DialBlockList)
	if err != nil {
		return nil, err
	}

	// Network Host with transport layer
	var mplexEnable bool = true
	var bwr metrics.Reporter = nil // Don't do statistics
	var tpt smux.Transport = makeSmuxTransport(mplexEnable)
	var host p2phost.Host
	host, err = core.DefaultHostOption(ctx, id, ps, bwr, fs, tpt)
	if err != nil {
		return nil, err
	}

	// DHT Protocol
	dstore := ds.NewMapDatastore()
	var client routing.IpfsRouting
	if config.ClientOnly {
		client = dht.NewDHTClient(ctx, host, dstore)
	} else {
		client = dht.NewDHT(ctx, host, dstore)
	}

	// Bitswap Protocol
	peerHost := rhost.Wrap(host, client)
	bitswapNetwork := bsnet.NewFromIpfsHost(peerHost, client)
	blockstore := NewPeerBlockstore(peerObj)
	exchange := bitswap.New(ctx, host.ID(), bitswapNetwork, blockstore, !config.RudeBitswap)

	net := &Network{
		client:   client,
		exchange: exchange,
		store:    blockstore,
	}
	return net, nil
}
