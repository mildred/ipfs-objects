// Package dht implements a distributed hash table that satisfies the ipfs routing
// interface. This DHT is modeled after kademlia with Coral and S/Kademlia modifications.
package dht

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	routing "gx/ipfs/QmZghcVHwXQC3Zvnvn24LgTmSPkEn2o3PDyKb6nrtPRzRh/go-libp2p-routing"
	pb "gx/ipfs/QmdFu71pRmWMNWht96ZTJ3wRx4D7BPJ2JfHH24z59Gidsc/go-libp2p-kad-dht/pb"
	providers "gx/ipfs/QmdFu71pRmWMNWht96ZTJ3wRx4D7BPJ2JfHH24z59Gidsc/go-libp2p-kad-dht/providers"

	ci "gx/ipfs/QmNiCwBNA8MWDADTFVq1BonUEJbS2SvjAoNkZZrhEwcuUi/go-libp2p-crypto"
	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	ds "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore"
	goprocess "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	goprocessctx "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/context"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	kb "gx/ipfs/QmUwZcbSVMsLZzovZssH96rCUM5FAkrjaqhHLhJnFYd5z3/go-libp2p-kbucket"
	cid "gx/ipfs/QmV5gPoRsjN1Gid3LMdNZTyfCtP2DsvqEbMAmz82RmmiGk/go-cid"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	record "gx/ipfs/QmZp9q8DbrGLztoxpkTC62mnRayRwHcAzGJJ8AvYRwjanR/go-libp2p-record"
	recpb "gx/ipfs/QmZp9q8DbrGLztoxpkTC62mnRayRwHcAzGJJ8AvYRwjanR/go-libp2p-record/pb"
	base32 "gx/ipfs/QmZvZSVtvxak4dcTkhsQhqd1SQ6rg5UzaSTu62WfWKjj93/base32"
	host "gx/ipfs/QmbzbRyd22gcW92U1rA2yKagB3myMYhk45XBknJ49F9XWJ/go-libp2p-host"
)

var log = logging.Logger("dht")

var ProtocolDHT protocol.ID = "/ipfs/kad/1.0.0"
var ProtocolDHTOld protocol.ID = "/ipfs/dht"

// NumBootstrapQueries defines the number of random dht queries to do to
// collect members of the routing table.
const NumBootstrapQueries = 5

// IpfsDHT is an implementation of Kademlia with Coral and S/Kademlia modifications.
// It is used to implement the base IpfsRouting module.
type IpfsDHT struct {
	host      host.Host        // the network services we need
	self      peer.ID          // Local peer (yourself)
	peerstore pstore.Peerstore // Peer Registry

	datastore ds.Datastore // Local data

	routingTable *kb.RoutingTable // Array of routing tables for differently distanced nodes
	providers    *providers.ProviderManager

	birth time.Time // When this peer started up

	Validator record.Validator // record validator funcs
	Selector  record.Selector  // record selection funcs

	ctx  context.Context
	proc goprocess.Process

	strmap map[peer.ID]*messageSender
	smlk   sync.Mutex
}

// NewDHT creates a new DHT object with the given peer as the 'local' host
func NewDHT(ctx context.Context, h host.Host, dstore ds.Batching) *IpfsDHT {
	dht := NewDHTClient(ctx, h, dstore)

	h.SetStreamHandler(ProtocolDHT, dht.handleNewStream)
	h.SetStreamHandler(ProtocolDHTOld, dht.handleNewStream)

	return dht
}

// NewDHTClient creates a new DHT object with the given peer as the 'local' host
func NewDHTClient(ctx context.Context, h host.Host, dstore ds.Batching) *IpfsDHT {
	dht := makeDHT(ctx, h, dstore)

	// register for network notifs.
	dht.host.Network().Notify((*netNotifiee)(dht))

	dht.proc = goprocessctx.WithContextAndTeardown(ctx, func() error {
		// remove ourselves from network notifs.
		dht.host.Network().StopNotify((*netNotifiee)(dht))
		return nil
	})

	dht.proc.AddChild(dht.providers.Process())

	dht.Validator["pk"] = record.PublicKeyValidator
	dht.Selector["pk"] = record.PublicKeySelector

	return dht
}

func makeDHT(ctx context.Context, h host.Host, dstore ds.Batching) *IpfsDHT {
	return &IpfsDHT{
		datastore:    dstore,
		self:         h.ID(),
		peerstore:    h.Peerstore(),
		host:         h,
		strmap:       make(map[peer.ID]*messageSender),
		ctx:          ctx,
		providers:    providers.NewProviderManager(ctx, h.ID(), dstore),
		birth:        time.Now(),
		routingTable: kb.NewRoutingTable(KValue, kb.ConvertPeerID(h.ID()), time.Minute, h.Peerstore()),

		Validator: make(record.Validator),
		Selector:  make(record.Selector),
	}
}

// putValueToPeer stores the given key/value pair at the peer 'p'
func (dht *IpfsDHT) putValueToPeer(ctx context.Context, p peer.ID,
	key string, rec *recpb.Record) error {

	pmes := pb.NewMessage(pb.Message_PUT_VALUE, key, 0)
	pmes.Record = rec
	rpmes, err := dht.sendRequest(ctx, p, pmes)
	switch err {
	case ErrReadTimeout:
		log.Warningf("read timeout: %s %s", p.Pretty(), key)
		fallthrough
	default:
		return err
	case nil:
		break
	}

	if err != nil {
		return err
	}

	if !bytes.Equal(rpmes.GetRecord().Value, pmes.GetRecord().Value) {
		return errors.New("value not put correctly")
	}
	return nil
}

var errInvalidRecord = errors.New("received invalid record")

// getValueOrPeers queries a particular peer p for the value for
// key. It returns either the value or a list of closer peers.
// NOTE: It will update the dht's peerstore with any new addresses
// it finds for the given peer.
func (dht *IpfsDHT) getValueOrPeers(ctx context.Context, p peer.ID, key string) (*recpb.Record, []pstore.PeerInfo, error) {

	pmes, err := dht.getValueSingle(ctx, p, key)
	if err != nil {
		return nil, nil, err
	}

	// Perhaps we were given closer peers
	peers := pb.PBPeersToPeerInfos(pmes.GetCloserPeers())

	if record := pmes.GetRecord(); record != nil {
		// Success! We were given the value
		log.Debug("getValueOrPeers: got value")

		// make sure record is valid.
		err = dht.verifyRecordOnline(ctx, record)
		if err != nil {
			log.Info("Received invalid record! (discarded)")
			// return a sentinal to signify an invalid record was received
			err = errInvalidRecord
			record = new(recpb.Record)
		}
		return record, peers, err
	}

	if len(peers) > 0 {
		log.Debug("getValueOrPeers: peers")
		return nil, peers, nil
	}

	log.Warning("getValueOrPeers: routing.ErrNotFound")
	return nil, nil, routing.ErrNotFound
}

// getValueSingle simply performs the get value RPC with the given parameters
func (dht *IpfsDHT) getValueSingle(ctx context.Context, p peer.ID, key string) (*pb.Message, error) {
	meta := logging.LoggableMap{
		"key":  key,
		"peer": p,
	}

	defer log.EventBegin(ctx, "getValueSingle", meta).Done()

	pmes := pb.NewMessage(pb.Message_GET_VALUE, key, 0)
	resp, err := dht.sendRequest(ctx, p, pmes)
	switch err {
	case nil:
		return resp, nil
	case ErrReadTimeout:
		log.Warningf("read timeout: %s %s", p.Pretty(), key)
		fallthrough
	default:
		return nil, err
	}
}

// getLocal attempts to retrieve the value from the datastore
func (dht *IpfsDHT) getLocal(key string) (*recpb.Record, error) {
	log.Debugf("getLocal %s", key)

	v, err := dht.datastore.Get(mkDsKey(key))
	if err != nil {
		return nil, err
	}
	log.Debugf("found %s in local datastore")

	byt, ok := v.([]byte)
	if !ok {
		return nil, errors.New("value stored in datastore not []byte")
	}
	rec := new(recpb.Record)
	err = proto.Unmarshal(byt, rec)
	if err != nil {
		return nil, err
	}

	err = dht.verifyRecordLocally(rec)
	if err != nil {
		log.Debugf("local record verify failed: %s (discarded)", err)
		return nil, err
	}

	return rec, nil
}

// getOwnPrivateKey attempts to load the local peers private
// key from the peerstore.
func (dht *IpfsDHT) getOwnPrivateKey() (ci.PrivKey, error) {
	sk := dht.peerstore.PrivKey(dht.self)
	if sk == nil {
		log.Warningf("%s dht cannot get own private key!", dht.self)
		return nil, fmt.Errorf("cannot get private key to sign record!")
	}
	return sk, nil
}

// putLocal stores the key value pair in the datastore
func (dht *IpfsDHT) putLocal(key string, rec *recpb.Record) error {
	data, err := proto.Marshal(rec)
	if err != nil {
		return err
	}

	return dht.datastore.Put(mkDsKey(key), data)
}

// Update signals the routingTable to Update its last-seen status
// on the given peer.
func (dht *IpfsDHT) Update(ctx context.Context, p peer.ID) {
	log.Event(ctx, "updatePeer", p)
	dht.routingTable.Update(p)
}

// FindLocal looks for a peer with a given ID connected to this dht and returns the peer and the table it was found in.
func (dht *IpfsDHT) FindLocal(id peer.ID) pstore.PeerInfo {
	p := dht.routingTable.Find(id)
	if p != "" {
		return dht.peerstore.PeerInfo(p)
	}
	return pstore.PeerInfo{}
}

// findPeerSingle asks peer 'p' if they know where the peer with id 'id' is
func (dht *IpfsDHT) findPeerSingle(ctx context.Context, p peer.ID, id peer.ID) (*pb.Message, error) {
	defer log.EventBegin(ctx, "findPeerSingle", p, id).Done()

	pmes := pb.NewMessage(pb.Message_FIND_NODE, string(id), 0)
	resp, err := dht.sendRequest(ctx, p, pmes)
	switch err {
	case nil:
		return resp, nil
	case ErrReadTimeout:
		log.Warningf("read timeout: %s %s", p.Pretty(), id)
		fallthrough
	default:
		return nil, err
	}
}

func (dht *IpfsDHT) findProvidersSingle(ctx context.Context, p peer.ID, key *cid.Cid) (*pb.Message, error) {
	defer log.EventBegin(ctx, "findProvidersSingle", p, key).Done()

	pmes := pb.NewMessage(pb.Message_GET_PROVIDERS, key.KeyString(), 0)
	resp, err := dht.sendRequest(ctx, p, pmes)
	switch err {
	case nil:
		return resp, nil
	case ErrReadTimeout:
		log.Warningf("read timeout: %s %s", p.Pretty(), key)
		fallthrough
	default:
		return nil, err
	}
}

// nearestPeersToQuery returns the routing tables closest peers.
func (dht *IpfsDHT) nearestPeersToQuery(pmes *pb.Message, count int) []peer.ID {
	closer := dht.routingTable.NearestPeers(kb.ConvertKey(pmes.GetKey()), count)
	return closer
}

// betterPeerToQuery returns nearestPeersToQuery, but iff closer than self.
func (dht *IpfsDHT) betterPeersToQuery(pmes *pb.Message, p peer.ID, count int) []peer.ID {
	closer := dht.nearestPeersToQuery(pmes, count)

	// no node? nil
	if closer == nil {
		return nil
	}

	// == to self? thats bad
	for _, p := range closer {
		if p == dht.self {
			log.Debug("attempted to return self! this shouldn't happen...")
			return nil
		}
	}

	var filtered []peer.ID
	for _, clp := range closer {
		// Dont send a peer back themselves
		if p == clp {
			continue
		}

		filtered = append(filtered, clp)
	}

	// ok seems like closer nodes
	return filtered
}

// Context return dht's context
func (dht *IpfsDHT) Context() context.Context {
	return dht.ctx
}

// Process return dht's process
func (dht *IpfsDHT) Process() goprocess.Process {
	return dht.proc
}

// Close calls Process Close
func (dht *IpfsDHT) Close() error {
	return dht.proc.Close()
}

func mkDsKey(s string) ds.Key {
	return ds.NewKey(base32.RawStdEncoding.EncodeToString([]byte(s)))
}