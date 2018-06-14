package dht

import (
	"net"
	"testing"

	"github.com/lbryio/reflector.go/dht/bits"
)

func TestBootstrapPing(t *testing.T) {
	b := NewBootstrapNode(bits.Rand(), 10, bootstrapDefaultRefreshDuration)

	listener, err := net.ListenPacket(network, "127.0.0.1:54320")
	if err != nil {
		panic(err)
	}

	if err := b.Connect(listener.(*net.UDPConn)); err != nil {
		t.Error(err)
	}
	defer b.Shutdown()

	b.Shutdown()
}
