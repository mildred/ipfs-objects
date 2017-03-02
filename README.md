
TODO
====

* find a local port that is free we can use to listen to. Because ipfs uses
  SO_REUSEADDR/SO_REUSEPORT, it can hijack an already establisked socket. Make
  sure we bind to a free socket. In the meantime, there is the `-listen`b CLI
  option.

* Rename ipfs-objects to ipobj-net

* Poviders should unique the list of returned peers

* GetRecordFrom should find peer if not in peerstore

* use direct records pushing for message passing
