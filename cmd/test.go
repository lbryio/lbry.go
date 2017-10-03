package cmd

import (
	"fmt"
	"sync"

	"github.com/lbryio/lbry.go/jsonrpc"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	var testCmd = &cobra.Command{
		Use:   "test",
		Short: "For testing stuff",
		Run:   test,
	}
	RootCmd.AddCommand(testCmd)
}

func test(cmd *cobra.Command, args []string) {
	daemon = jsonrpc.NewClient("")
	addresses, err := daemon.WalletList()
	if err != nil {
		panic(err)
	} else if addresses == nil || len(*addresses) == 0 {
		panic(fmt.Errorf("could not find an address in wallet"))
	}
	claimAddress = (*addresses)[0]
	if claimAddress == "" {
		panic(fmt.Errorf("found blank claim address"))
	}

	var wg sync.WaitGroup

	publishes := []jsonrpc.PublishOptions{
		{
			Title:        strPtr("a"),
			Language:     strPtr("en"),
			ClaimAddress: &claimAddress,
			ChannelName:  strPtr("@x"),
		},
		{
			Title:        strPtr("b"),
			Language:     strPtr("en"),
			ClaimAddress: &claimAddress,
			ChannelName:  strPtr("@x"),
		},
	}

	for _, o := range publishes {
		wg.Add(1)
		go func(o jsonrpc.PublishOptions) {
			defer wg.Done()

			log.Println("Publishing " + *o.Title)
			response, err := daemon.Publish(*o.Title, "/home/grin/Desktop/cake.jpg", 0.01, o)
			if err != nil {
				spew.Dump([]interface{}{o, err})
			}
			spew.Dump(response)
		}(o)
	}

	wg.Wait()
}
