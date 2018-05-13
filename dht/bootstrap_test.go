package dht

import (
	"net"
	"testing"
)

func TestBootstrapPing(t *testing.T) {
	b := NewBootstrapNode(RandomBitmapP(), 10, bootstrapDefaultRefreshDuration)

	listener, err := net.ListenPacket(network, "127.0.0.1:54320")
	if err != nil {
		panic(err)
	}

	b.Connect(listener.(*net.UDPConn))
	defer b.Shutdown()

	b.Shutdown()
}
