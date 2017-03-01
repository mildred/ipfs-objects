export GOPATH=$(PWD)

ipfs-objects: always
	go build -o $@ ./src/cmd/$@

gx-undo:
	cd src/github.com/ipfs/go-ipfs; gx-go rewrite --undo
	cd src/github.com/ipfs/go-ipld-node; gx-go rewrite --undo
	cd src/github.com/ipfs/go-ipld-cbor; gx-go rewrite --undo

always:
.PHONY: always
