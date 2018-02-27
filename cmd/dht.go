package cmd

import (
	"strconv"
	"time"

	"github.com/lbryio/lbry.go/dht"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	d := &cobra.Command{
		Use:   "dht <action>",
		Args:  cobra.ExactArgs(1),
		Short: "Do DHT things",
		Run:   dhtCmd,
	}
	RootCmd.AddCommand(d)

	ping := &cobra.Command{
		Use:   "ping <ip>",
		Args:  cobra.ExactArgs(1),
		Short: "Ping a node on the DHT",
		Run:   dhtPingCmd,
	}
	d.AddCommand(ping)
}

func dhtCmd(cmd *cobra.Command, args []string) {
	log.Errorln("chose a command")
}

func dhtPingCmd(cmd *cobra.Command, args []string) {
	//ip := args[0]

	port := 49449 // + (rand.Int() % 10)

	config := dht.NewStandardConfig()
	config.Address = "127.0.0.1:" + strconv.Itoa(port)
	config.PrimeNodes = []string{
		"127.0.0.1:10001",
	}

	d := dht.New(config)
	log.Println("Starting...")
	go d.Run()

	time.Sleep(2 * time.Second)

	for {
		peers, err := d.FindNode("012b66fc7052d9a0c8cb563b8ede7662003ba65f425c2661b5c6919d445deeb31469be8b842d6faeea3f2b3ebcaec845")
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}

		log.Println("Found peers:", peers)
		break
	}

	log.Println("done")
}
