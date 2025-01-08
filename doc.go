// uproxy is a trivial user-space proxy for Linux.  Currently, it supports TCP on IPv4 only.  The primary package is uprodxy/proxy,
// which contains the core proxying logic.  The cmd/tproxy package contains a simple command-line interface for the proxy.  This is intended to
// be used as a transparent proxy using the tproxy facility of the Linux kernel.  See https://docs.kernel.org/networking/tproxy.html for
// more information.  The README.md file in the repository contains more information on how to use the proxy and how to configure tproxy.
package uproxy
