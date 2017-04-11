## Synopsis

Peer-to-peer, random dialer telephone system realization on [Enduro/X cluster middleware for C/C++/Go/Python](http://www.endurox.org).
The project uses Enduro/X XATMI distributed processing and clustering to join in single cluster domain all involed nodes (in this
case Raspberry PI machines). Which offers services like PHONE, RING, PLAYBACK (busy, wait), MIC. The system reads the switch state
picked up, or hanged up the phone and does the random call to other nodes. If other node picks up the phone
the call is established and user can speak over.



