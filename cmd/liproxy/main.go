// liproxy implements a trivial TCP userspace proxy that is intended to receive connections redirected using linux liproxy.  See the README.md in the parent package (uproxy)
// for more information.
//
// liproxy will bind by default to 127.0.0.1:9091.  This can be overridden by passing the -bindAddr flag.  The insertionLabel flag can be used to add a label to the
// each read chunk of each flow.  When inserted into the data stream, the label is appended to the end of each read chunk, followed by a newline.  This is probably
// only useful if the client and server are sending only a trivial amount (meaning <= 1 segement-worth) of data in each flow.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/blorticus-go/uproxy/proxy"
)

func die(logger *slog.Logger, f string, a ...any) {
	if logger != nil {
		logger.Error(f, a...)
	} else {
		fmt.Fprintf(os.Stderr, f, a...)
	}
	os.Exit(1)
}

func dieIfError(err error, logger *slog.Logger) {
	if err != nil {
		die(logger, "%s", err.Error())
	}
}

func main() {
	cliArgs := ParseCommandLineArguments()

	if cliArgs.BindAddressIsInvalid() {
		die(nil, "Invalid bind address (%s).  Expect <ip>:<port>", cliArgs.BindAddress)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger.Info("starting")

	proxy := proxy.New()

	err := proxy.BindTo(cliArgs.BindAddress)
	dieIfError(err, logger)

	logger.Info("bound", "address", cliArgs.BindAddress)

	if cliArgs.InsertionLabel != "" {
		terminatedLabel := cliArgs.InsertionLabel + "\n"
		proxy.OnRead(func(b []byte) []byte {
			return append([]byte(terminatedLabel), b...)
		})
	}

	go proxy.StartProxying()

	logger.Info("accepting incoming flows")
	loopOnProxyMessages(proxy.MessageChannel(), logger)

	logger.Info("terminating")
	proxy.Terminate()
}

func loopOnProxyMessages(msgChannel <-chan *proxy.Message, logger *slog.Logger) {
	signalsChannel := make(chan os.Signal, 5)
	signal.Notify(signalsChannel, syscall.SIGABRT, syscall.SIGHUP, syscall.SIGINT)

	for {
		select {
		case s := <-signalsChannel:
			logger.Info("received signal", "signal", s.String())
			return

		case proxyMsg := <-msgChannel:
			switch proxyMsg.Type {
			case proxy.ReceivedIncomingClientsideFlow:
				logger.Info("incoming clientside flow", "clientsideAddr", proxyMsg.LocalAddr.String(), "serversideAddr", proxyMsg.RemoteAddr.String())

			case proxy.InitiatingOutgoingServersideFlow:
				logger.Info("initiating serverside flow", "clientsideAddr", proxyMsg.LocalAddr.String(), "serversideAddr", proxyMsg.RemoteAddr.String())

			case proxy.ServerClosedFlow:
				logger.Info("server closed flow", "clientsideAddr", proxyMsg.LocalAddr.String(), "serversideAddr", proxyMsg.RemoteAddr.String())

			case proxy.ClientClosedFlow:
				logger.Info("client closed flow", "clientsideAddr", proxyMsg.LocalAddr.String(), "serversideAddr", proxyMsg.RemoteAddr.String())

			case proxy.ListenError:
				logger.Error("listen error", "errorMsg", proxyMsg.Error.Error())
				return

			case proxy.ProxyError:
				logger.Warn("proxy error", "clientsideAddr", proxyMsg.LocalAddr.String(), "serversideAddr", proxyMsg.RemoteAddr.String(), "errorMsg", proxyMsg.Error.Error())
			}
		}
	}
}
