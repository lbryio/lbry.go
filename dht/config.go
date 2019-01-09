package dht

import (
	"strconv"
	"time"

	"github.com/lbryio/reflector.go/dht/bits"
	peerproto "github.com/lbryio/reflector.go/peer"
)

const (
	Network     = "udp4"
	DefaultPort = 4444

	DefaultAnnounceRate   = 10               // send at most this many announces per second
	DefaultReannounceTime = 50 * time.Minute // should be a bit less than hash expiration time

	// TODO: all these constants should be defaults, and should be used to set values in the standard Config. then the code should use values in the config
	// TODO: alternatively, have a global Config for constants. at least that way tests can modify the values
	alpha           = 3             // this is the constant alpha in the spec
	bucketSize      = 8             // this is the constant k in the spec
	nodeIDLength    = bits.NumBytes // bytes. this is the constant B in the spec
	messageIDLength = 20            // bytes.

	udpRetry            = 1
	udpTimeout          = 5 * time.Second
	udpMaxMessageLength = 4096 // bytes. I think our longest message is ~676 bytes, so I rounded up to 1024
	//                            scratch that. a findValue could return more than K results if a lot of nodes are storing that value, so we need more buffer

	maxPeerFails = 3 // after this many failures, a peer is considered bad and will be removed from the routing table
	//tExpire     = 60 * time.Minute // the time after which a key/value pair expires; this is a time-to-live (TTL) from the original publication date
	tRefresh = 1 * time.Hour // the time after which an otherwise unaccessed bucket must be refreshed
	//tReplicate   = 1 * time.Hour    // the interval between Kademlia replication events, when a node is required to publish its entire database
	//tNodeRefresh = 15 * time.Minute // the time after which a good node becomes questionable if it has not messaged us

	compactNodeInfoLength = nodeIDLength + 6 // nodeID + 4 for IP + 2 for port

	tokenSecretRotationInterval = 5 * time.Minute // how often the token-generating secret is rotated
)

// Config represents the configure of dht.
type Config struct {
	// this node's address. format is `ip:port`
	Address string
	// the seed nodes through which we can join in dht network
	SeedNodes []string
	// the hex-encoded node id for this node. if string is empty, a random id will be generated
	NodeID string
	// print the state of the dht every X time
	PrintState time.Duration
	// the port that clients can use to download blobs using the LBRY peer protocol
	PeerProtocolPort int
	// if nonzero, an RPC server will listen to requests on this port and respond to them
	RPCPort int
	// the time after which the original publisher must reannounce a key/value pair
	ReannounceTime time.Duration
	// send at most this many announces per second
	AnnounceRate int
	// channel that will receive notifications about announcements
	AnnounceNotificationCh chan announceNotification
}

// NewStandardConfig returns a Config pointer with default values.
func NewStandardConfig() *Config {
	return &Config{
		Address: "0.0.0.0:" + strconv.Itoa(DefaultPort),
		SeedNodes: []string{
			"lbrynet1.lbry.io:4444",
			"lbrynet2.lbry.io:4444",
			"lbrynet3.lbry.io:4444",
		},
		PeerProtocolPort: peerproto.DefaultPort,
		ReannounceTime:   DefaultReannounceTime,
		AnnounceRate:     DefaultAnnounceRate,
	}
}
