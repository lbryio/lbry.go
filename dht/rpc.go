package dht

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/lbryio/lbry.go/dht/bits"
	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/gorilla/mux"
	rpc2 "github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
)

type rpcReceiver struct {
	dht *DHT
}

type RpcPingArgs struct {
	Address string
}

func (rpc *rpcReceiver) Ping(r *http.Request, args *RpcPingArgs, result *string) error {
	if args.Address == "" {
		return errors.Err("no address given")
	}

	err := rpc.dht.Ping(args.Address)
	if err != nil {
		return err
	}

	*result = pingSuccessResponse
	return nil
}

type RpcFindArgs struct {
	Key    string
	NodeID string
	IP     string
	Port   int
}

func (rpc *rpcReceiver) FindNode(r *http.Request, args *RpcFindArgs, result *[]Contact) error {
	key, err := bits.FromHex(args.Key)
	if err != nil {
		return err
	}

	toQuery, err := bits.FromHex(args.NodeID)
	if err != nil {
		return err
	}

	c := Contact{ID: toQuery, IP: net.ParseIP(args.IP), Port: args.Port}
	req := Request{Method: findNodeMethod, Arg: &key}

	nodeResponse := rpc.dht.node.Send(c, req)
	if nodeResponse != nil && nodeResponse.Contacts != nil {
		*result = nodeResponse.Contacts
	}
	return nil
}

type RpcFindValueResult struct {
	Contacts []Contact
	Value    string
}

func (rpc *rpcReceiver) FindValue(r *http.Request, args *RpcFindArgs, result *RpcFindValueResult) error {
	key, err := bits.FromHex(args.Key)
	if err != nil {
		return err
	}
	toQuery, err := bits.FromHex(args.NodeID)
	if err != nil {
		return err
	}
	c := Contact{ID: toQuery, IP: net.ParseIP(args.IP), Port: args.Port}
	req := Request{Arg: &key, Method: findValueMethod}

	nodeResponse := rpc.dht.node.Send(c, req)
	if nodeResponse != nil && nodeResponse.FindValueKey != "" {
		*result = RpcFindValueResult{Value: nodeResponse.FindValueKey}
		return nil
	}
	if nodeResponse != nil && nodeResponse.Contacts != nil {
		*result = RpcFindValueResult{Contacts: nodeResponse.Contacts}
		return nil
	}

	return errors.Err("not sure what happened")
}

type RpcIterativeFindValueArgs struct {
	Key string
}

type RpcIterativeFindValueResult struct {
	Contacts   []Contact
	FoundValue bool
}

func (rpc *rpcReceiver) IterativeFindValue(r *http.Request, args *RpcIterativeFindValueArgs, result *RpcIterativeFindValueResult) error {
	key, err := bits.FromHex(args.Key)
	if err != nil {
		return err
	}
	foundContacts, found, err := FindContacts(rpc.dht.node, key, false, nil)
	if err != nil {
		return err
	}
	result.Contacts = foundContacts
	result.FoundValue = found
	return nil
}

type RpcBucketResponse struct {
	Start       string
	End         string
	NumContacts int
	Contacts    []Contact
}

type RpcRoutingTableResponse struct {
	NodeID     string
	NumBuckets int
	Buckets    []RpcBucketResponse
}

func (rpc *rpcReceiver) GetRoutingTable(r *http.Request, args *struct{}, result *RpcRoutingTableResponse) error {
	result.NodeID = rpc.dht.node.id.String()
	result.NumBuckets = len(rpc.dht.node.rt.buckets)
	for _, b := range rpc.dht.node.rt.buckets {
		result.Buckets = append(result.Buckets, RpcBucketResponse{
			Start:       b.Range.Start.String(),
			End:         b.Range.End.String(),
			NumContacts: b.Len(),
			Contacts:    b.Contacts(),
		})
	}
	return nil
}

func (rpc *rpcReceiver) AddKnownNode(r *http.Request, args *Contact, result *string) error {
	rpc.dht.node.AddKnownNode(*args)
	return nil
}

func (dht *DHT) runRPCServer(port int) {
	addr := "0.0.0.0:" + strconv.Itoa(port)

	s := rpc2.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	err := s.RegisterService(&rpcReceiver{dht: dht}, "rpc")
	if err != nil {
		log.Error(errors.Prefix("registering rpc service", err))
		return
	}

	handler := mux.NewRouter()
	handler.Handle("/", s)
	server := &http.Server{Addr: addr, Handler: handler}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("RPC server listening on %s", addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Error(err)
		}
	}()

	<-dht.grp.Ch()
	err = server.Shutdown(context.Background())
	if err != nil {
		log.Error(errors.Prefix("shutting down rpc service", err))
		return
	}
	wg.Wait()
}
