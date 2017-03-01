package main

import (
	"flag"
	"fmt"
	"strings"

	ma "github.com/multiformats/go-multiaddr"
)

type Config struct {
	ListenAddrs ListenAddrs
}

func (cfg *Config) Flags(f *flag.FlagSet) {
	f.Var(&cfg.ListenAddrs, "listen", "List of address to listen to")
}

type ListenAddrs []string

func (i *ListenAddrs) String() string {
	return strings.Join(*i, ",")
}

func (i *ListenAddrs) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ListenAddrs) Get() ([]ma.Multiaddr, error) {
	var list []string = *i
	var listen []ma.Multiaddr

	if len(list) == 0 {
		list = []string{
			"/ip4/0.0.0.0/tcp/4001",
			"/ip4/0.0.0.0/udp/4002/utp",
			"/ip6/::/tcp/4001",
			"/ip6/::/udp/4002/utp",
		}
	}

	for _, addr := range list {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("Failure to parse listen address %s: %s", addr, err.Error())
		}
		listen = append(listen, maddr)
	}
	return listen, nil
}
