package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
)

func main() {
	dhcp := &DHCPServer{}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		dhcp.Shutdown()
	}()

	var wg sync.WaitGroup
	var m sync.Mutex
	var errs []error

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dhcp.Start("0.0.0.0:67")
		if err != nil {
			m.Lock()
			errs = append(errs, err)
			m.Unlock()
		}
	}()
	wg.Wait()

	if len(errs) > 0 {
		for _, err := range errs {
			log.Printf("[ERROR] %v", err)
		}
		os.Exit(1)
	}
}
