package ipobj

import (
	"context"
	"errors"
	"io"

	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// Content identifier
// could be content identifiers:
// - a multihash
// - hash in another format
// or an indirect reference (impleneted as direct or indirect ?):
// - public key fingerprint
// - any other record system

const RecordCidCode = 0x0220
const RecordMultihashCode = 0x00

func NewRecordCid(key string) *cid.Cid {
	return cid.NewCidV1(RecordCidCode, []byte(key))
}

func NewRecordObjAddr(key string) ObjAddr {
	h, e := mh.Encode([]byte(key), RecordMultihashCode)
	if e != nil {
		panic(e)
	}
	return ObjAddr(cid.NewCidV1(RecordCidCode, h).Bytes())
}

type ObjAddr []byte

// Multuaddress or node info
type PeerAddr []byte

// Peer info
type PeerInfo struct {
	Id    []byte
	Addrs []PeerAddr
}

// Record
type Record struct {
	PeerId  []byte
	Content []byte
}

type Network interface {
	Id() []byte

	// List providers for ObjHash. if updated is true, omit address of providers
	// that explicitely don't try to maintain the object up to date.
	Providers(ctx context.Context, obj ObjAddr) (<-chan []PeerInfo, error)

	// Get an object obj
	GetObject(ctx context.Context, obj ObjAddr) (io.Reader, error)

	// Get a record, a record generally contains a address to the object it
	// resolves and a version number to be able to order the records
	GetRecord(ctx context.Context, record string) <-chan *Record

	// Get a record, a record generally contains a address to the object it
	// resolves and a version number to be able to order the records
	GetRecordFrom(ctx context.Context, peerId []byte, key string) ([]byte, error)

	// Advertise the posession or not of an object.
	ProvideObject(ctx context.Context, obj ObjAddr, provide bool) error

	// Advertise a record on the DHT.
	// Good practice is to GetRecord before so we can update ourselves
	ProvideRecord(ctx context.Context, key string, rec []byte) error

	// Tell a peer that its record is not up to date
	UpdatePeerRecord(ctw context.Context, peerId []byte, key string, record []byte) error
}

type Peer interface {
	// The network tells us that peer has an updated version of obj. We should
	// update if we told the network so.
	Updated(peer PeerAddr, obj ObjAddr)

	// The network requires the object
	GetObject(obj ObjAddr) (io.Reader, error)

	// The network requires a record
	GetRecord(key string) ([]byte, error)

	// The network tells us to update our record
	NewRecord(key string, value []byte, peer []byte)
}

var NoObject error = errors.New("No object for NullPeer")
var NullPeer Peer = &NullPeerType{}

type NullPeerType struct{}

func (*NullPeerType) Updated(peer PeerAddr, obj ObjAddr) {}
func (*NullPeerType) GetObject(obj ObjAddr) (io.Reader, error) {
	return nil, NoObject
}
func (*NullPeerType) GetRecord(key string) ([]byte, error) {
	return nil, nil
}
func (*NullPeerType) NewRecord(key string, value []byte, peer []byte) {}
