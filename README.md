# uproxy

A trivial linux userspace proxy.

## Overview

`uproxy` provides a package (`uproxy/proxy`) that implements a simple userspace proxy for IPv4 TCP connections.
Currently, the proxy is intended to be the target for [a linux tproxy rule](https://docs.kernel.org/networking/tproxy.html).
It uses the [IP_TRANSPARENT sockopt](https://man7.org/linux/man-pages/man7/ip.7.html) on incoming socket connections.  This
allows the proxy to see the original source IP and port, as well as the original destination IP and port, for that connection.
When the three-way handshake completes for the incoming connection, the proxy initiates an outgoing connection (also using
IP_TRANSPARENT) to the original destination IP:port, and sets the source IP:port to match the original connection.

At this point, the two connections (the documentation calls them _flows_) are bound together.  The connection that was
forwarded to the proxy is called the `incoming` flow, while the connection created by the proxy is called the `outgoing`
flow.  Notice that both connections have the same `srcip`:`srcport`:`dstip`:`dstport` tuple. Once both flows are connected,
any data that arrives on the incoming is copied to the outgoing, and vice-versa.

The data can be modified in-flight using an `interceptor` function which is what makes a proxy of this type useful at all.

The incoming flow cannot be sourced using the same kernel instance as the proxy executes on.

## tproxy Configuration

The following provides an [iptables](https://www.netfilter.org/projects/iptables/index.html) example configuration to
redirect flows transparently to a uproxy instance.

```bash
# 1
sudo iptables -t mangle -N DIVERT
sudo iptables -t mangle -A PREROUTING -i ens6 -p tcp -m socket --dport 9090 -j DIVERT
sudo iptables -t mangle -A PREROUTING -i ens7 -p tcp -m socket --sport 9090 -j DIVERT
sudo iptables -t mangle -A DIVERT -j MARK --set-mark 1
sudo iptables -t mangle -A DIVERT -j ACCEPT
# 2
sudo iptables -t mangle -A PREROUTING -i ens6 -p tcp --dport 9090 -j TPROXY --tproxy-mark 1 --on-port 9091 --on-ip 127.0.0.1
sudo iptables -t mangle -A PREROUTING -i ens7 -p tcp --sport 9090 -j TPROXY --tproxy-mark 1 --on-port 9091 --on-ip 127.0.0.1
# 3
sudo ip rule add fwmark 1 lookup 100
sudo ip route add local 0.0.0.0/0 dev lo table 100
```

This example assumes that incoming flows will all arrive on the interface `ens6` and the default route
points to a nexthop reachable on the `ens7` interface.  Furthermore, these rules only intercept TCP traffic bound for
port 9090.

For clarity, it is better to start at #2.  The first of these two rules matches packets incoming to the interface `ens6` that
are TCP with a destination port of 9090.  On a match, the packet is redirected to TPROXY.  The packet is [marked](https://devops-insider.mygraphql.com/zh-cn/latest/kernel/network/netfilter/mark.html)
with the value 1, and the proxy receiver for the resulting socket is bound to `127.0.0.1:9091`.

The rules at #3 intercept packets marked with 1 and redirect them to the [ip table](https://www.kernel.org/doc/Documentation/networking/policy-routing.txt)
numbered 100.  This table has a single route entry, which binds `0.0.0.0/0` to the loopback interface.  This allows the outgoing flow to use any
source address, which is needed so that it can match the source address of the incoming flow.

Returning back to #1, this set of rules is a shortcut.  #2 matched initial packets that do not result in socket data.  This importantly includes the three-way
handshake.  However, once the `TPROXY` action is used, the socket becomes transparent.  We still need the normal data segements and their underlying packets to
be carried to route table 100 (so that the proxy can continue to generate packets with the masqueraded source address), but these packets don't need to go
through the `TPROXY` action (to my knowledge, it works if they do go through the `TPROXY` action, but it is less efficient).  The [socket](https://ipset.netfilter.org/iptables-extensions.man.html)
iptables module is used to match socket-bound packets.

Putting this together, the first command creates a new iptables [mangle chain](https://www.digitalocean.com/community/tutorials/a-deep-dive-into-iptables-and-netfilter-architecture#the-mangle-table)
called `DIVERT`.

The next two rules direct TCP packets matched by the `socket` module with the TCP source port 9090 (if it is arriving on the incoming) or destination port 9090 
(if it is arriving on the outgoing) and redirects them to the `DIVERT` table.  (This must be done in `PREROUTING`, which is why locally-sourced connections cannot be directed to the proxy).

In the `DIVERT` table, each packet is marked.  The `ACCEPT` action accepts the packet for normal route processing, and importantly, stops the iptables preroute
processing.  This is the optimization discussed above.

## Sample implementation

The sub-module `cmd/liproxy` implements a simple proxy using `uproxy`.  It assumes things are set up as described in the last section.  It binds to a port,
accepts incoming flows, creates matching outgoing flows, proxies traffic between the flows, and optional appends a label to each block of socket read data.
