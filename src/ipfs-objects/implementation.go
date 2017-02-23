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

	p2phost "gx/ipfs/QmbzbRyd22gcW92U1rA2yKagB3myMYhk45XBknJ49F9XWJ/go-libp2p-host"
	dht "gx/ipfs/QmdFu71pRmWMNWht96ZTJ3wRx4D7BPJ2JfHH24z59Gidsc/go-libp2p-kad-dht"
	ds "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore"
	mamask "gx/ipfs/QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv/multiaddr-filter"
	metrics "gx/ipfs/QmPj6rmE2sWJ65h6b8F4fcN5kySDhYqL2Ty8DWWF3WEUNS/go-libp2p-metrics"
	routing "gx/ipfs/QmZghcVHwXQC3Zvnvn24LgTmSPkEn2o3PDyKb6nrtPRzRh/go-libp2p-routing"
	rhost "gx/ipfs/QmSNJRX4uphb3Eyp69uYbpRVvgqjPxfjnJmjcdMWkDH5Pn/go-libp2p/p2p/host/routed"
	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	smux "gx/ipfs/QmeZBgYBHvxMukGK5ojg28BCNLB9SeXqT7XXg6o7r2GbJy/go-stream-muxer"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	ic "gx/ipfs/QmNiCwBNA8MWDADTFVq1BonUEJbS2SvjAoNkZZrhEwcuUi/go-libp2p-crypto"
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
