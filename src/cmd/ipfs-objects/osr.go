package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	osr "ipobj-osr"

	ic "github.com/libp2p/go-libp2p-crypto"
)

func genosr(args []string) error {
	var f flag.FlagSet
	var keyfile string
	var order uint64
	var output string
	var salt string
	f.StringVar(&keyfile, "k", "", "Secret key file")
	f.StringVar(&output, "o", "", "Output file")
	f.StringVar(&salt, "s", "", "Salt")
	f.Uint64Var(&order, "n", uint64(time.Now().Unix()), "Record order")
	f.Parse(args[1:])

	var err error
	var sk ic.PrivKey
	if keyfile == "" {
		sk, err = dummySecretKey()
	} else {
		sk, err = readKeyFile(keyfile)
	}

	var rec osr.Record = osr.Record{
		CID:   f.Arg(0),
		Order: order,
		Salt:  salt,
	}

	path, err := osr.Path(salt, sk.GetPublic())
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Generated record: /iprs%s\n", path)

	data, err := rec.Encode(sk)
	if err != nil {
		return err
	}

	out := os.Stdout
	if output != "" {
		out, err = os.Create(output)
		if err != nil {
			return err
		}
		defer out.Close()
	}

	_, err = out.Write(data)
	if err != nil {
		return err
	}

	return nil
}
