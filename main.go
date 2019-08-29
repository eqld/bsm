package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eqld/bsm/consumer"
	"github.com/eqld/bsm/framebuffer"
	"github.com/eqld/bsm/supplier"
)

const (
	defaultBufferFrameSize = 1024 * 1024
	defaultBufferFrames    = 1024
	defaultListenInput     = "unix:///tmp/bytehub.sock"
	defaultListenOutput    = "tcp://0.0.0.0:4096"
)

var (
	bufferFrameSize = flag.Int("buffer-frame-size", defaultBufferFrameSize, "size of buffer frame in bytes")
	bufferFrames    = flag.Int("buffer-frames", defaultBufferFrames, "amount of buffer frames")
	listenInput     = flag.String("listen-input", defaultListenInput, "protocol ('tcp', 'tcp4', 'tcp6', 'unix' or 'unixpacket') and address to listen for input stream suppliers")
	listenOutput    = flag.String("listen-output", defaultListenOutput, "protocol ('tcp', 'tcp4', 'tcp6', 'unix' or 'unixpacket') and address to listen for output stream consumers")
)

func parseNetAddr(netAddr string) (net, addr string, err error) {
	p := strings.Index(netAddr, "://")
	if p < 0 {
		err = fmt.Errorf("malformed address: %s", netAddr)
		return
	}
	net, addr = netAddr[:p], netAddr[p+3:]
	return
}

func main() {
	flag.Parse()

	log.Printf(
		"service started listening at %s for input stream suppliers and at %s for outut stream consumers, buffer size is %d frames of %d bytes (%d bytes total)\n",
		*listenInput, *listenOutput, *bufferFrames, *bufferFrameSize, *bufferFrames**bufferFrameSize,
	)

	// handle system signals

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		select {
		case <-ctx.Done():
		case signal := <-signals:
			log.Printf("received signal '%v', terminating\n", signal)
		}
	}()

	// create services

	suppNet, suppAddr, err := parseNetAddr(*listenInput)
	if err != nil {
		log.Printf("fail parse address to listen for supplier connections: %v\n", err)
		return
	}

	consNet, consAddr, err := parseNetAddr(*listenOutput)
	if err != nil {
		log.Printf("fail parse address to listen for consumer connections: %v\n", err)
		return
	}

	listenerSupplier, err := net.Listen(suppNet, suppAddr)
	if err != nil {
		log.Printf("fail to listen for input stream suppliers at %s: %v\n", *listenInput, err)
		return
	}
	defer listenerSupplier.Close()

	listenerConsumer, err := net.Listen(consNet, consAddr)
	if err != nil {
		log.Printf("fail to listen for output stream consumers at %s: %v\n", *listenOutput, err)
		return
	}
	defer listenerConsumer.Close()

	buffer := framebuffer.New(*bufferFrameSize, *bufferFrames)
	defer buffer.Close()

	frames := make(chan framebuffer.Frame, *bufferFrames)
	supp := supplier.New(listenerSupplier, buffer, frames)
	cons := consumer.New(listenerConsumer, frames)

	// start doing things

	go func() {
		defer cancel()
		supp.Serve(ctx)
	}()

	go func() {
		defer cancel()
		cons.Serve(ctx)
	}()

	go func() {
		defer cancel()
		tick := time.Tick(3 * time.Second)
		for {
			select {
			case <-tick:
				stat := buffer.Stat()
				log.Printf(
					"[heartbeat] buffer usage: %d / %d (%d %%)\n",
					stat.Used, stat.Total, stat.Used*100/stat.Total,
				)
			case <-ctx.Done():
				return
			}
		}
	}()

	// wait for termination

	<-ctx.Done()
	log.Println("service terminated")
}
