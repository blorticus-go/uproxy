# liproxy

A trivial userspace transparent proxy for TCP flows.

## Synopsis

```bash
liproxy [-bindAddress <ip>:<port>] [-insertionLabel <label>]
```

## Description

`liproxy` implements a trivial userspace transparent proxy using [uproxy](https://github.com/blorticus-go/uproxy).  It is designed to be used with the linux TPROXY facility (see the `uproxy` README.md for more details).

`liproxy` binds to a listen socket and awaits incoming TCP conenctions.  When a connection arrives, after the completion of the three-way handshake, it initiates an outbound connection to the same destination as the incoming connection, and sets the source address and port to match the incoming connection.  All data that arrives from the incoming flow is copied to the outgoing flow, and vice-versa.  If an `insertionLabel` is provided, then on each socket read, the _label_ value (followed by a newline, i.e., UTF-8 \n) is appended to the data read before it is proxied to the reverse flow.

## Defaults

The default `bindAddress` is `127.0.0.1:9090`.  The default `insertionLabel` is the empty string, which means that no label is inserted.