package ipfs_objects

import (
	"context"
	"ipobj"

	blocks "github.com/ipfs/go-ipfs/blocks"
	blockstore "github.com/ipfs/go-ipfs/blocks/blockstore"

	cid "gx/ipfs/QmcTcsTvfaeEBRFo1TkFgT8sRmgi1n1LTZpecfVP8fzpGD/go-cid"
)

var _ blockstore.Blockstore = &PeerBlockstore{}

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

func (pb *PeerBlockstore) DeleteBlock(*cid.Cid) error {
	return nil // Cannot force to delete block
}

func (pb *PeerBlockstore) Has(id *cid.Cid) (bool, error) {
	return pb.list[string(id.Bytes())], nil
}

func (pb *PeerBlockstore) Get(id *cid.Cid) (blocks.Block, error) {
	key := id.Bytes()
	reader, err := pb.peer.GetObject(key)
	if err != nil {
		return nil, err
	}
	bytes, err := readerToBytes(reader)
	if err != nil {
		return nil, err
	}
	return blocks.NewBlockWithCid(bytes, id)
}

func (pb *PeerBlockstore) Put(blocks.Block) error {
	return nil // Cannot force to accept block
}

func (pb *PeerBlockstore) PutMany(blks []blocks.Block) error {
	for _, blk := range blks {
		err := pb.Put(blk)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pb *PeerBlockstore) AllKeysChan(ctx context.Context) (<-chan *cid.Cid, error) {
	res := make(chan *cid.Cid)
	go func() {
		for id := range pb.list {
			key, err := cid.Cast([]byte(id))
			if err != nil {
				panic(err)
			}
			res <- key
		}
	}()
	return res, nil
}
