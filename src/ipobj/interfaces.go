package ipobj

import (
	"context"
	"errors"
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

// Peer info
type PeerInfo struct {
	Id    string
	Addrs []PeerAddr
}

// Record
type Record struct {
	PeerId  string
	Content []byte
}

type Network interface {
	// List providers for ObjHash. if updated is true, omit address of providers
	// that explicitely don't try to maintain the object up to date.
	// The search continues until the result channel is closed.
	Providers(ctx context.Context, obj ObjAddr, updated bool) (<-chan []PeerInfo, error)

	// Get an object obj
	GetObject(ctx context.Context, obj ObjAddr) (io.Reader, error)

	// Get a record, a record generally contains a address to the object it
	// resolves and a version number to be able to order the records
	GetRecord(ctx context.Context, record string) <-chan *Record

	// Get a record, a record generally contains a address to the object it
	// resolves and a version number to be able to order the records
	GetRecordFrom(ctx context.Context, peerId string, key string) ([]byte, error)

	// Advertise the posession or not of an object.
	ProvideObject(ctx context.Context, obj ObjAddr, provide bool) error

	// Advertise a record on the DHT.
	// Good practice is to GetRecord before so we can update ourselves
	ProvideRecord(ctx context.Context, key string, rec []byte) error

	// Tell a peer that its record is not up to date
	UpdatePeerRecord(ctw context.Context, peerId string, key string, record []byte) error
}

type Peer interface {
	// The network tells us that peer has an updated version of obj. We should
	// update if we told the network so.
	Updated(peer PeerAddr, obj ObjAddr)

	// The network requires the object
	GetObject(obj ObjAddr) (io.Reader, error)
}

var NoObject error = errors.New("No object for NullPeer")
var NullPeer Peer = &nullPeer{}

type nullPeer struct{}

func (*nullPeer) Updated(peer PeerAddr, obj ObjAddr) {}
func (*nullPeer) GetObject(obj ObjAddr) (io.Reader, error) {
	return nil, NoObject
}
