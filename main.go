package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"golang.org/x/sync/errgroup"
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

	var g errgroup.Group

	g.Go(func() error { return dhcp.Start("0.0.0.0:67") })
	g.Go(func() error { return tftp.Start("0.0.0.0:69") })
	err := g.Wait()
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
}
