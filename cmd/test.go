package cmd

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

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
	var wg sync.WaitGroup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-c
		log.Println("got signal")
	}()
	log.Println("waiting for ctrl+c")
	wg.Wait()
	log.Println("done waiting")
}
