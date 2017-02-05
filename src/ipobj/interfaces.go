package ipobj

import (
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
	Id      PeerId
	Address []PeerAddr
}

type Network interface {
	// List providers for ObjHash. if updated is true, omit address of providers
	// that explicitely don't try to maintain the object up to date.
	// The search continues until the result channel is closed.
	Providers(obj ObjAddr, updated bool) <-chan []PeerInfo

	// Get an object obj provided by peer at address
	GetObject(peer PeerAddr, obj ObjAddr) io.Reader

	// Advertise the posession of an object. update to true means we are willing
	// to update if a new version can be found.
	Provide(obj ObjAddr, update bool)

	// Tell peer that we have an updated version of object
	Update(peer PeerAddr, obj ObjAddr)
}

type Peer interface {
	// The network tells us that peer has an updated version of obj. We should
	// update if we told the network so.
	Updated(peer PeerAddr, obj ObjAddr)
}
