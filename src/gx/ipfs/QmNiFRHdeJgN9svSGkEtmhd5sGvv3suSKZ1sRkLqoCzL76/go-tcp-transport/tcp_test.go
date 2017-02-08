package tcp

import (
	"testing"

	ma "gx/ipfs/QmUAQaWbKxGCUTuoQVvvicbQNZ9APF5pDGWyAZSe93AtKH/go-multiaddr"
	utils "gx/ipfs/QmWMia2fBVBesMerbtApQY7Tj2sgTaziveBACfCRUcv45f/go-libp2p-transport/test"
)

func TestTcpTransport(t *testing.T) {
	ta := NewTCPTransport()
	tb := NewTCPTransport()

	zero := "/ip4/127.0.0.1/tcp/0"
	utils.SubtestTransport(t, ta, tb, zero)
}

func TestTcpTransportCantListenUtp(t *testing.T) {
	utpa, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/0/utp")
	if err != nil {
		t.Fatal(err)
	}

	tpt := NewTCPTransport()
	_, err = tpt.Listen(utpa)
	if err == nil {
		t.Fatal("shouldnt be able to listen on utp addr with tcp transport")
	}
}
