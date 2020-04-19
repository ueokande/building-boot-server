package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
)

var (
	flgIPXEForPXEPath   = flag.String("ipxe-for-pxe-path", "", "Path to iPXE ROM for iPXE (undionly.kpxe).  Download it from http://boot.ipxe.org/undionly.kpxe and specify the local path")
	flgPXEPathPrefix    = flag.String("pxe-path-prefix", "/usr/lib/syslinux/bios", "Path prefix where pxe images are contained in")
	flgKernelPathPrefix = flag.String("kernel-path-prefix", "/boot", "Path prefix where kernel images are contained in")
)

func main() {
	flag.Parse()

	dhcp := &DHCPServer{
		BootFilename: "pxelinux/pxelinux.0",
	}
	tftp := &TFTPServer{
		IPXERomPath:      *flgIPXEForPXEPath,
		PXEPathPrefix:    *flgPXEPathPrefix,
		KernelPathPrefix: *flgKernelPathPrefix,
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		dhcp.Shutdown()
		tftp.Shutdown()
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := tftp.Start("0.0.0.0:69")
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
