IPFS-Objects
============

IPFS-Objects is a simple command line tool that uses the complete IPFS code base to do simple things. There is no daemon running and no global configuration. You can define the settings needed for each command as you run it. This is still a work in progress, but this tool is designed to make it easy to share mutable hierarchies of files on top of the IPFS network.

It contains the logic to make updates using OSR (ordered Signed Records) and the code can be reused for any other kind of record.

There are few sub-commands available:

- keygen: generate a key, required for most other operations
- gen-osr: generates an OSR for a new version of the record
- advertise: advertise a particular OSR
- resolve: watch the network for the most recent OSR

What is this Ordered Signed Record
----------------------------------

This is a piece of data that contains:

- a payload string that should contain a [CID](https://github.com/ipfs/go-cid)
- a version number, it's quite practical to put a timestamp there
- a public key
- some salt
- a cryptographic signature

Given two records, it is possible to determine if the records are valid and which record is the most up to date. It can allow mutable values within the IPFS network.

The salt is used so we can use the same key pair to generate multiple mutable records. Two records with the same salt can be compared and ordered. If the salt is different, the two records are not supposed to represent the same thing, and thus will not be compared.

Given a public key and a salt, the record generates a unique CID of the form `/osr/<key fingerprint><salt>`

How advertisement works?
------------------------

Advertisements of these records is special. It uses the DHT part of IPFS. Each record is given a CID of the form `/osr/<hash>` and it is placed on the DHT along with the informations about the peer that is advertising the record.

The DHT allows anyone to query for the location of an object given its CID. To look for a CID, a peer just needs to query the DHT for the OSR. It finds a list of peers that advertise a version of the CID (not the most up to date necessarily) and it will then have to ask each one of these peers for the record value. Multiple records are compared and the most up to date is used.

A good behaviour for a listener is to then tell each of the peers that advertised old versions of the record to update to the last up to date version.


Build
=====

- clone recursive
- `make gx-undo`
- `make`


Use
===

Prepare the client, server and record keys:

    ./ipfs-objects keygen -o record.key
    ./ipfs-objects keygen -o client.key
    ./ipfs-objects keygen -o server.key

Generate the OSR record:

    ./ipfs-objects gen-osr -o test1.osr -k record.key TEST

On one terminal, advertise for the record:

    ./ipfs-objects advertise -k server.key -t 1m test1.osr

Remember the record key starting with `/iprs/osr` and usr it for the next
command.  On another terminal, ask for the record (Ctrl-C to stop):

    ./ipfs-objects -listen /ip4/0.0.0.0/tcp/5000 resolve -k client.key /iprs/osr/...

Generate a new version of the record:

    ./ipfs-objects gen-osr -o test2.osr -k record.key TEST

Update the record on outdated peers (Ctrl-C to stop):

    ./ipfs-objects -listen /ip4/0.0.0.0/tcp/5000 update -k client.key test2.osr

Check the new record is available (Ctrl-C to stop):

    ./ipfs-objects -listen /ip4/0.0.0.0/tcp/5000 resolve -k client.key /iprs/osr/...


TODO
====

* find a local port that is free we can use to listen to. Because ipfs uses
  SO_REUSEADDR/SO_REUSEPORT, it can hijack an already establisked socket. Make
  sure we bind to a free socket. In the meantime, there is the `-listen`b CLI
  option.

* GetRecordFrom should find peer if not in peerstore

* use direct records pushing for message passing


Hacking
=======

This is still a work in progress, and I have many things to do that prevent me from working on this as much as I want to. However, if you want to look at the code, this is not a usual golang repository. Instead, it is designed to have the GOPATH pointing to the repository checkout itself. The `src` directory contains both the code and vendored code.

- `src/cmd/ipfs-objects`: the command line
- `src/ipobj`: go interfaces to implement
- `src/ipobj-osr`: OSR data object
- `src/ipobj-net`: Glue code that implements the interface in `ipobj` and links to the IPFS code base.
- `src/simpleipc`: IPC code that I plan to use later

The complete IPFS network is abstracted behind an interface. For the moment it is a Go interface, but in the future, I want this to be a separated process communicating with an IPC mechanism featuring zero-copy (implemented in `src/simpleipc`). The idea is to have a frontend with a command line that implements the record (OSR in this case) and have it call the backend for network communications.

This will allow anyone to extend IPFS with any kind of record without having to reimplement everything in their own language, or having to link with the Go code. They'll just have to call a secondary process and communicate with it using unix domain sockets to perform network operations.
