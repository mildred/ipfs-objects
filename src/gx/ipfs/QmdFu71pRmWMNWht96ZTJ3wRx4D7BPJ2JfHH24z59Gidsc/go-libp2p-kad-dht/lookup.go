package dht

import (
	"context"

	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	kb "gx/ipfs/QmUwZcbSVMsLZzovZssH96rCUM5FAkrjaqhHLhJnFYd5z3/go-libp2p-kbucket"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	pset "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer/peerset"
	notif "gx/ipfs/QmZghcVHwXQC3Zvnvn24LgTmSPkEn2o3PDyKb6nrtPRzRh/go-libp2p-routing/notifications"
)

// Required in order for proper JSON marshaling
func pointerizePeerInfos(pis []pstore.PeerInfo) []*pstore.PeerInfo {
	out := make([]*pstore.PeerInfo, len(pis))
	for i, p := range pis {
		np := p
		out[i] = &np
	}
	return out
}

func loggableKey(k string) logging.LoggableMap {
	return logging.LoggableMap{
		"key": k,
	}
}

// Kademlia 'node lookup' operation. Returns a channel of the K closest peers
// to the given key
func (dht *IpfsDHT) GetClosestPeers(ctx context.Context, key string) (<-chan peer.ID, error) {
	e := log.EventBegin(ctx, "getClosestPeers", loggableKey(key))
	tablepeers := dht.routingTable.NearestPeers(kb.ConvertKey(key), KValue)
	if len(tablepeers) == 0 {
		return nil, kb.ErrLookupFailure
	}

	out := make(chan peer.ID, KValue)
	peerset := pset.NewLimited(KValue)

	for _, p := range tablepeers {
		select {
		case out <- p:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		peerset.Add(p)
	}

	// since the query doesnt actually pass our context down
	// we have to hack this here. whyrusleeping isnt a huge fan of goprocess
	parent := ctx
	query := dht.newQuery(key, func(ctx context.Context, p peer.ID) (*dhtQueryResult, error) {
		// For DHT query command
		notif.PublishQueryEvent(parent, &notif.QueryEvent{
			Type: notif.SendingQuery,
			ID:   p,
		})

		closer, err := dht.closerPeersSingle(ctx, key, p)
		if err != nil {
			log.Debugf("error getting closer peers: %s", err)
			return nil, err
		}

		var filtered []pstore.PeerInfo
		for _, clp := range closer {
			if peerset.TryAdd(clp) {
				select {
				case out <- clp:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				filtered = append(filtered, dht.peerstore.PeerInfo(clp))
			}
		}

		// For DHT query command
		notif.PublishQueryEvent(parent, &notif.QueryEvent{
			Type:      notif.PeerResponse,
			ID:        p,
			Responses: pointerizePeerInfos(filtered),
		})

		return &dhtQueryResult{closerPeers: filtered}, nil
	})

	go func() {
		defer close(out)
		defer e.Done()
		// run it!
		_, err := query.Run(ctx, tablepeers)
		if err != nil {
			log.Debugf("closestPeers query run error: %s", err)
		}
	}()

	return out, nil
}

func (dht *IpfsDHT) closerPeersSingle(ctx context.Context, key string, p peer.ID) ([]peer.ID, error) {
	pmes, err := dht.findPeerSingle(ctx, p, peer.ID(key))
	if err != nil {
		return nil, err
	}

	var out []peer.ID
	for _, pbp := range pmes.GetCloserPeers() {
		pid := peer.ID(pbp.GetId())
		if pid != dht.self { // dont add self
			dht.peerstore.AddAddrs(pid, pbp.Addresses(), pstore.TempAddrTTL)
			out = append(out, pid)
		}
	}
	return out, nil
}
