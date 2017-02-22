package ipfs_objects

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	mssmux "gx/ipfs/QmRVYfZ7tWNHPBzWiG6KWGzvT2hcGems8srihsQE29x1U5/go-smux-multistream"
	spdy "gx/ipfs/QmWUNsat6Jb19nC5CiJCDXepTkxjdxi3eZqeoB6mrmmaGu/go-smux-spdystream"
	mplex "gx/ipfs/QmbPSoTDH3JFKkSxsCQDyVMFT76KcQz7ELWEetyyg3aeDA/go-smux-multiplex"
	yamux "gx/ipfs/Qmbn7RYyWzBVXiUp9jZ1dA4VADHy9DtS7iZLwfhEUQvm3U/go-smux-yamux"
	smux "gx/ipfs/QmeZBgYBHvxMukGK5ojg28BCNLB9SeXqT7XXg6o7r2GbJy/go-stream-muxer"
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
