package ipfs_objects

import (
	"context"
	"fmt"
	"ipobj"
	"net"

	exchange "github.com/ipfs/go-ipfs/exchange"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	ipfs_bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"
	//dht "github.com/libp2p/go-libp2p-kad-dht"
	//routing "github.com/libp2p/go-libp2p-routing"

	ic "gx/ipfs/QmNiCwBNA8MWDADTFVq1BonUEJbS2SvjAoNkZZrhEwcuUi/go-libp2p-crypto"
	metrics "gx/ipfs/QmPj6rmE2sWJ65h6b8F4fcN5kySDhYqL2Ty8DWWF3WEUNS/go-libp2p-metrics"
	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	ds "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore"
	mamask "gx/ipfs/QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv/multiaddr-filter"
	p2pbhost "gx/ipfs/QmSNJRX4uphb3Eyp69uYbpRVvgqjPxfjnJmjcdMWkDH5Pn/go-libp2p/p2p/host/basic"
	ipfs_rhost "gx/ipfs/QmSNJRX4uphb3Eyp69uYbpRVvgqjPxfjnJmjcdMWkDH5Pn/go-libp2p/p2p/host/routed"
	swarm "gx/ipfs/QmY8hduizbuACvYmL4aZQbpFeKhEQJ1Nom2jY6kv6rL8Gf/go-libp2p-swarm"
	ipfs_peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	routing "gx/ipfs/QmZghcVHwXQC3Zvnvn24LgTmSPkEn2o3PDyKb6nrtPRzRh/go-libp2p-routing"
	p2phost "gx/ipfs/QmbzbRyd22gcW92U1rA2yKagB3myMYhk45XBknJ49F9XWJ/go-libp2p-host"
	dht "gx/ipfs/QmdFu71pRmWMNWht96ZTJ3wRx4D7BPJ2JfHH24z59Gidsc/go-libp2p-kad-dht"
	smux "gx/ipfs/QmeZBgYBHvxMukGK5ojg28BCNLB9SeXqT7XXg6o7r2GbJy/go-stream-muxer"
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

// isolates the complex initialization steps
func constructPeerHost(ctx context.Context, id peer.ID, ps pstore.Peerstore, bwr metrics.Reporter, fs []*net.IPNet, tpt smux.Transport) (p2phost.Host, error) {

	// no addresses to begin with. we'll start later.
	swrm, err := swarm.NewSwarmWithProtector(ctx, nil, id, ps, nil, tpt, bwr)
	if err != nil {
		return nil, err
	}

	network := (*swarm.Network)(swrm)

	for _, f := range fs {
		network.Swarm().Filters.AddDialFilter(f)
	}

	host := p2pbhost.New(network, p2pbhost.NATPortMap, bwr)

	return host, nil
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
	host, err = constructPeerHost(ctx, id, ps, bwr, fs, tpt)
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
	peerHost := ipfs_rhost.Wrap(host, client)
	bitswapNetwork := ipfs_bsnet.NewFromIpfsHost(peerHost, client)
	blockstore := NewPeerBlockstore(peerObj)
	exchange := bitswap.New(ctx, ipfs_peer.ID(host.ID()), bitswapNetwork, blockstore, !config.RudeBitswap)

	net := &Network{
		client:   client,
		exchange: exchange,
		store:    blockstore,
	}
	return net, nil
}
