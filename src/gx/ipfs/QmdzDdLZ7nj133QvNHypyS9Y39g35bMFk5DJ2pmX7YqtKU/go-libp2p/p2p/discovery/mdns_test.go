package discovery

import (
	"context"
	"testing"
	"time"

	bhost "gx/ipfs/QmdzDdLZ7nj133QvNHypyS9Y39g35bMFk5DJ2pmX7YqtKU/go-libp2p/p2p/host/basic"

	netutil "gx/ipfs/QmPS1HTBHiJcqxDAZ4s8bGt22HtL3oC67TPR3BsrvM44Z1/go-libp2p-netutil"
	host "gx/ipfs/QmPsRtodRuBUir32nz5v4zuSBTSszrR1d3fA6Ahb6eaejj/go-libp2p-host"

	pstore "gx/ipfs/QmeXj9VAjmYQZxpmVz7VzccbJrpmr8qkCDSjfVNsPTWTYU/go-libp2p-peerstore"
)

type DiscoveryNotifee struct {
	h host.Host
}

func (n *DiscoveryNotifee) HandlePeerFound(pi pstore.PeerInfo) {
	n.h.Connect(context.Background(), pi)
}

func TestMdnsDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := bhost.New(netutil.GenSwarmNetwork(t, ctx))
	b := bhost.New(netutil.GenSwarmNetwork(t, ctx))

	sa, err := NewMdnsService(ctx, a, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	sb, err := NewMdnsService(ctx, b, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	_ = sb

	n := &DiscoveryNotifee{a}

	sa.RegisterNotifee(n)

	time.Sleep(time.Second * 2)

	err = a.Connect(ctx, pstore.PeerInfo{ID: b.ID()})
	if err != nil {
		t.Fatal(err)
	}
}
