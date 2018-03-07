package dht

import (
	"net"
	"strconv"
	"strings"
	"time"
)

type UDPConn interface {
	ReadFromUDP([]byte) (int, *net.UDPAddr, error)
	WriteToUDP([]byte, *net.UDPAddr) (int, error)
	SetWriteDeadline(time.Time) error
}

type testUDPPacket struct {
	data []byte
	addr *net.UDPAddr
}

type testUDPConn struct {
	addr   *net.UDPAddr
	toRead chan testUDPPacket
	writes chan testUDPPacket
}

func newTestUDPConn(addr string) *testUDPConn {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		panic("addr needs ip and port")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}
	return &testUDPConn{
		addr:   &net.UDPAddr{IP: net.IP(parts[0]), Port: port},
		toRead: make(chan testUDPPacket),
		writes: make(chan testUDPPacket),
	}
}

func (t testUDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	select {
	case packet := <-t.toRead:
		n := copy(b, packet.data)
		return n, packet.addr, nil
		//default:
		//	return 0, nil, nil
	}
}

func (t testUDPConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	t.writes <- testUDPPacket{data: b, addr: addr}
	return len(b), nil
}

func (t testUDPConn) SetWriteDeadline(tm time.Time) error {
	return nil
}
