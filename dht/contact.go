package dht

import (
	"bytes"
	"encoding/json"
	"net"
	"sort"
	"strconv"

	"github.com/lbryio/lbry.go/v3/dht/bits"

	"github.com/cockroachdb/errors"
	"github.com/lyoshenka/bencode"
)

// TODO: if routing table is ever empty (aka the node is isolated), it should re-bootstrap

// Contact contains information for contacting another node on the network
type Contact struct {
	ID       bits.Bitmap
	IP       net.IP
	Port     int // the udp port used for the dht
	PeerPort int // the tcp port a peer can be contacted on for blob requests
}

// Equals returns true if two contacts are the same.
func (c Contact) Equals(other Contact, checkID bool) bool {
	return c.IP.Equal(other.IP) && c.Port == other.Port && (!checkID || c.ID == other.ID)
}

// Addr returns the address of the contact.
func (c Contact) Addr() *net.UDPAddr {
	return &net.UDPAddr{IP: c.IP, Port: c.Port}
}

// String returns a short string representation of the contact
func (c Contact) String() string {
	str := c.ID.HexShort() + "@" + c.Addr().String()
	if c.PeerPort != 0 {
		str += "(" + strconv.Itoa(c.PeerPort) + ")"
	}
	return str
}

func (c Contact) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(&struct {
		ID       string
		IP       string
		Port     int
		PeerPort int
	}{
		ID:       c.ID.Hex(),
		IP:       c.IP.String(),
		Port:     c.Port,
		PeerPort: c.PeerPort,
	})
	return b, errors.Wrap(err, "")
}

// MarshalCompact returns a compact byteslice representation of the contact
// NOTE: The compact representation always uses the tcp PeerPort, not the udp Port. This is dumb, but that's how the python daemon does it
func (c Contact) MarshalCompact() ([]byte, error) {
	if c.IP.To4() == nil {
		return nil, errors.Wrap(errors.New("ip not set"), "")
	}
	if c.PeerPort < 0 || c.PeerPort > 65535 {
		return nil, errors.Wrap(errors.New("invalid port"), "")
	}

	var buf bytes.Buffer
	buf.Write(c.IP.To4())
	buf.WriteByte(byte(c.PeerPort >> 8))
	buf.WriteByte(byte(c.PeerPort))
	buf.Write(c.ID[:])

	if buf.Len() != compactNodeInfoLength {
		return nil, errors.Wrap(errors.New("i dont know how this happened"), "")
	}

	return buf.Bytes(), nil
}

// UnmarshalCompact unmarshals the compact byteslice representation of a contact.
// NOTE: The compact representation always uses the tcp PeerPort, not the udp Port. This is dumb, but that's how the python daemon does it
func (c *Contact) UnmarshalCompact(b []byte) error {
	if len(b) != compactNodeInfoLength {
		return errors.Wrap(errors.New("invalid compact length"), "")
	}
	c.IP = net.IPv4(b[0], b[1], b[2], b[3]).To4()
	c.PeerPort = int(uint16(b[5]) | uint16(b[4])<<8)
	c.ID = bits.FromBytesP(b[6:])
	return nil
}

// MarshalBencode returns the serialized byte slice representation of a contact.
func (c Contact) MarshalBencode() ([]byte, error) {
	b, err := bencode.EncodeBytes([]interface{}{c.ID, c.IP.String(), c.Port})
	return b, errors.Wrap(err, "")
}

// UnmarshalBencode unmarshals the serialized byte slice into the appropriate fields of the contact.
func (c *Contact) UnmarshalBencode(b []byte) error {
	var raw []bencode.RawMessage
	err := bencode.DecodeBytes(b, &raw)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if len(raw) != 3 {
		return errors.Wrap(errors.Newf("contact must have 3 elements; got %d", len(raw)), "")
	}

	err = bencode.DecodeBytes(raw[0], &c.ID)
	if err != nil {
		return errors.Wrap(err, "")
	}

	var ipStr string
	err = bencode.DecodeBytes(raw[1], &ipStr)
	if err != nil {
		return errors.Wrap(err, "")
	}
	c.IP = net.ParseIP(ipStr).To4()
	if c.IP == nil {
		return errors.Wrap(errors.New("invalid IP"), "")
	}

	return bencode.DecodeBytes(raw[2], &c.Port)
}

func sortByDistance(contacts []Contact, target bits.Bitmap) {
	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].ID.Xor(target).Cmp(contacts[j].ID.Xor(target)) < 0
	})
}
