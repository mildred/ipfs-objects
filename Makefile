export GOPATH=$(PWD)

ipfs-objects: always
	go build -o $@ ./src/cmd/$@

gx-undo:
	cd src/github.com/ipfs/go-ipfs; gx-go rewrite --undo

always:
.PHONY: always
