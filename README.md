Build
=====

- clone recursive
- `make gx-undo`
- `make gx`


Use
===

Prepare the server key:

    ./ipfs-objects keygen -o advertise.key

On one terminal, advertise for a record:

    ./ipfs-objects advertise -k advertise.key -t 1m /hello/world " test "

On another terminal, ask for the record:

    ./ipfs-objects -listen /ip4/0.0.0.0/tcp/5000 resolve /hello world


TODO
====

* find a local port that is free we can use to listen to. Because ipfs uses
  SO_REUSEADDR/SO_REUSEPORT, it can hijack an already establisked socket. Make
  sure we bind to a free socket. In the meantime, there is the `-listen`b CLI
  option.

* GetRecordFrom should find peer if not in peerstore

* use direct records pushing for message passing
