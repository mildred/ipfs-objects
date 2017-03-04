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

    ./ipfs-objects gen-osr -o test.osr -k record.key TEST

On one terminal, advertise for the record:

    ./ipfs-objects advertise -k server.key -t 1m test.osr

Remember the record key starting with `/iprs/osr` and usr it for the next
command.  On another terminal, ask for the record:

    ./ipfs-objects -listen /ip4/0.0.0.0/tcp/5000 resolve -k client.key /iprs/osr/...


TODO
====

* find a local port that is free we can use to listen to. Because ipfs uses
  SO_REUSEADDR/SO_REUSEPORT, it can hijack an already establisked socket. Make
  sure we bind to a free socket. In the meantime, there is the `-listen`b CLI
  option.

* GetRecordFrom should find peer if not in peerstore

* use direct records pushing for message passing
