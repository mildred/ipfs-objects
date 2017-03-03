package net

import (
	"context"
	"ipobj"

	ipfs_cid "github.com/ipfs/go-cid"
	ipfs_blocks "github.com/ipfs/go-ipfs/blocks"
	ipfs_blockstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
)

var _ ipfs_blockstore.Blockstore = &PeerBlockstore{}
var _ dht.RecordHandler = &PeerRecord{}

type PeerRecord struct {
	peer ipobj.Peer
}

func (pr *PeerRecord) GetRecord(key string) ([]byte, error) {
	return pr.peer.GetRecord(key)
}

func (pr *PeerRecord) NewRecord(key string, value []byte, p peer.ID) bool {
	pr.peer.NewRecord(key, value, []byte(p))
	return true
}

type PeerBlockstore struct {
	peer ipobj.Peer
	list map[string]bool
}

func NewPeerBlockstore(peerObj ipobj.Peer) *PeerBlockstore {
	return &PeerBlockstore{
		peer: peerObj,
		list: map[string]bool{},
	}
}

func (pb *PeerBlockstore) DeleteBlock(*ipfs_cid.Cid) error {
	return nil // Cannot force to delete block
}

func (pb *PeerBlockstore) Has(id *ipfs_cid.Cid) (bool, error) {
	return pb.list[string(id.Bytes())], nil
}

func (pb *PeerBlockstore) Get(id *ipfs_cid.Cid) (ipfs_blocks.Block, error) {
	key := id.Bytes()
	reader, err := pb.peer.GetObject(key)
	if err != nil {
		return nil, err
	}
	bytes, err := readerToBytes(reader)
	if err != nil {
		return nil, err
	}
	return ipfs_blocks.NewBlockWithCid(bytes, id)
}

func (pb *PeerBlockstore) Put(ipfs_blocks.Block) error {
	return nil // Cannot force to accept block
}

func (pb *PeerBlockstore) PutMany(blks []ipfs_blocks.Block) error {
	for _, blk := range blks {
		err := pb.Put(blk)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pb *PeerBlockstore) AllKeysChan(ctx context.Context) (<-chan *ipfs_cid.Cid, error) {
	res := make(chan *ipfs_cid.Cid)
	go func() {
		for id := range pb.list {
			key, err := ipfs_cid.Cast([]byte(id))
			if err != nil {
				panic(err)
			}
			res <- key
		}
	}()
	return res, nil
}
