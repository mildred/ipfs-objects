package ipfs_objects

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	smux "github.com/jbenet/go-stream-muxer"
	mplex "github.com/whyrusleeping/go-smux-multiplex"
	mssmux "github.com/whyrusleeping/go-smux-multistream"
	spdy "github.com/whyrusleeping/go-smux-spdystream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

func makeSmuxTransport(mplexExp bool) smux.Transport {
	mstpt := mssmux.NewBlankTransport()

	ymxtpt := &yamux.Transport{
		AcceptBacklog:          8192,
		ConnectionWriteTimeout: time.Second * 10,
		KeepAliveInterval:      time.Second * 30,
		EnableKeepAlive:        true,
		MaxStreamWindowSize:    uint32(1024 * 512),
		LogOutput:              ioutil.Discard,
	}

	mstpt.AddTransport("/yamux/1.0.0", ymxtpt)

	mstpt.AddTransport("/spdy/3.1.0", spdy.Transport)

	if mplexExp {
		mstpt.AddTransport("/mplex/6.7.0", mplex.DefaultTransport)
	}

	// Allow muxer preference order overriding
	if prefs := os.Getenv("LIBP2P_MUX_PREFS"); prefs != "" {
		mstpt.OrderPreference = strings.Fields(prefs)
	}

	return mstpt
}
