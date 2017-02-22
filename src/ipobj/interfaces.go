package ipobj

import (
	"context"
	"io"
)

// Content identifier
// could be content identifiers:
// - a multihash
// - hash in another format
// or an indirect reference (impleneted as direct or indirect ?):
// - public key fingerprint
// - any other record system
type ObjAddr []byte

// Multuaddress or node info
type PeerAddr []byte

// Peer ID
type PeerId []byte

// Peer info
type PeerInfo struct {
	Id    PeerId
	Addrs []PeerAddr
}

// Record
type Record struct {
	PeerId  PeerId
	Content []byte
}

type Network interface {
	// List providers for ObjHash. if updated is true, omit address of providers
	// that explicitely don't try to maintain the object up to date.
	// The search continues until the result channel is closed.
	Providers(ctx context.Context, obj ObjAddr, updated bool) (<-chan []PeerInfo, error)

	// Get an object obj
	GetObject(ctx context.Context, obj ObjAddr) io.Reader

	// Get a record, a record generally contains a address to the object it
	// resolves and a version number to be able to order the records
	GetRecord(ctx context.Context, obj ObjAddr) <-chan Record

	// Advertise the posession of an object.
	ProvideObject(obj ObjAddr)

	// Advertise a record on the DHT.
	// Good practice is to GetRecord before so we can update ourselves
	ProvideRecord(obj ObjAddr, objs []Record)
}

type Peer interface {
	// The network tells us that peer has an updated version of obj. We should
	// update if we told the network so.
	Updated(peer PeerAddr, obj ObjAddr)
}
