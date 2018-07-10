package dht

import (
	"net"
	"net/http"
	"errors"
	"sync"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/lbryio/reflector.go/dht/bits"
)

type NodeRPCServer struct {
	Wg sync.WaitGroup
	Node *BootstrapNode
}

var mut sync.Mutex
var rpcServer *NodeRPCServer

type NodeRPC int

type PingArgs struct {
	NodeID string
	IP string
	Port int
}

type PingResult string


func (n *NodeRPC) Ping(r *http.Request, args *PingArgs, result *PingResult) error {
	if rpcServer == nil {
		return errors.New("no node set up")
	}
	toQuery, err := bits.FromHex(args.NodeID)
	if err != nil {
		return err
	}
	c := Contact{ID: toQuery, IP: net.ParseIP(args.IP), Port: args.Port}
	req := Request{Method: "ping"}
	nodeResponse := rpcServer.Node.Send(c, req)
	if nodeResponse != nil {
		*result = PingResult(nodeResponse.Data)
	}
	return nil
}

type FindArgs struct {
	Key string
	NodeID string
	IP string
	Port int
}

type ContactResponse struct {
	NodeID string
	IP string
	Port int
}

type FindNodeResult []ContactResponse

func (n *NodeRPC) FindNode(r *http.Request, args *FindArgs, result *FindNodeResult) error {
	if rpcServer == nil {
		return errors.New("no node set up")
	}
	key, err := bits.FromHex(args.Key)
	if err != nil {
		return err
	}
	toQuery, err := bits.FromHex(args.NodeID)
	if err != nil {
		return err
	}
	c := Contact{ID: toQuery, IP: net.ParseIP(args.IP), Port: args.Port}
	req := Request{ Arg: &key, Method: "findNode"}
	nodeResponse := rpcServer.Node.Send(c, req)
	contacts := []ContactResponse{}
	if nodeResponse != nil && nodeResponse.Contacts != nil {
		for _, foundContact := range nodeResponse.Contacts {
			contacts = append(contacts, ContactResponse{foundContact.ID.Hex(), foundContact.IP.String(), foundContact.Port})
		}
	}
	*result = FindNodeResult(contacts)
	return nil
}

type FindValueResult struct {
	Contacts []ContactResponse
	Value string
}

func (n *NodeRPC) FindValue(r *http.Request, args *FindArgs, result *FindValueResult) error {
	if rpcServer == nil {
		return errors.New("no node set up")
	}
	key, err := bits.FromHex(args.Key)
	if err != nil {
		return err
	}
	toQuery, err := bits.FromHex(args.NodeID)
	if err != nil {
		return err
	}
	c := Contact{ID: toQuery, IP: net.ParseIP(args.IP), Port: args.Port}
	req := Request{ Arg: &key, Method: "findValue"}
	nodeResponse := rpcServer.Node.Send(c, req)
	contacts := []ContactResponse{}
	if nodeResponse != nil && nodeResponse.FindValueKey != "" {
		*result = FindValueResult{Value: nodeResponse.FindValueKey}
		return nil
	} else if nodeResponse != nil && nodeResponse.Contacts != nil {
		for _, foundContact := range nodeResponse.Contacts {
			contacts = append(contacts, ContactResponse{foundContact.ID.Hex(), foundContact.IP.String(), foundContact.Port})
		}
		*result = FindValueResult{Contacts: contacts}
		return nil
	}
	return errors.New("not sure what happened")
}

type BucketResponse struct {
	Start string
	End string
	Count int
	Contacts []ContactResponse
}

type RoutingTableResponse struct {
	Count int
	Buckets []BucketResponse
}

type GetRoutingTableArgs struct {}

func (n *NodeRPC) GetRoutingTable(r *http.Request, args *GetRoutingTableArgs, result *RoutingTableResponse) error {
	if rpcServer == nil {
		return errors.New("no node set up")
	}
	result.Count = len(rpcServer.Node.rt.buckets)
	for _, b := range rpcServer.Node.rt.buckets {
		bucketInfo := []ContactResponse{}
		for _, c := range b.Contacts() {
			bucketInfo = append(bucketInfo, ContactResponse{c.ID.String(), c.IP.String(), c.Port})
		}
		result.Buckets = append(result.Buckets, BucketResponse{
			Start: b.Range.Start.String(), End: b.Range.End.String(), Contacts: bucketInfo,
			Count: b.Len(),
		})
	}
	return nil
}

type GetNodeIDArgs struct {}

type GetNodeIDResult string

func (n *NodeRPC) GetNodeID(r *http.Request, args *GetNodeIDArgs, result *GetNodeIDResult) error {
	if rpcServer == nil {
		return errors.New("no node set up")
	}
	log.Println("get node id")
	*result = GetNodeIDResult(rpcServer.Node.id.String())
	return nil
}

type PrintBucketInfoArgs struct {}

type PrintBucketInfoResult string

func (n *NodeRPC) PrintBucketInfo(r *http.Request, args *PrintBucketInfoArgs, result *PrintBucketInfoResult) error {
	if rpcServer == nil {
		return errors.New("no node set up")
	}
	rpcServer.Node.rt.printBucketInfo()
	*result = PrintBucketInfoResult("printed")
	return nil
}

func RunRPCServer(address, rpcPath string, node *BootstrapNode) NodeRPCServer {
	mut.Lock()
	defer mut.Unlock()
	rpcServer = &NodeRPCServer{
		Wg: sync.WaitGroup{},
		Node: node,
	}
	c := make(chan *http.Server)
	rpcServer.Wg.Add(1)
	go func() {
		s := rpc.NewServer()
		s.RegisterCodec(json.NewCodec(), "application/json")
		s.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
		node := new(NodeRPC)
		s.RegisterService(node, "")
		r := mux.NewRouter()
		r.Handle(rpcPath, s)
		server := &http.Server{Addr: address, Handler: r}
		log.Println("rpc listening on " + address)
		server.ListenAndServe()
		c <- server
	}()
	go func() {
		rpcServer.Wg.Wait()
		close(c)
		log.Println("stopped rpc listening on " + address)
		for server := range c {
			server.Close()
		}
	}()
	return *rpcServer
}
