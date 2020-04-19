package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"golang.org/x/sync/errgroup"
)

var (
	flgPXEBootFile = flag.String("pxe-boot-file", "pxelinux/pxelinux.0", "The file name used in PXE boot mode")
	flgTFTPBootDir = flag.String("tftp-boot-dir", "./tftpboot", "The directory including PXE images")
	flgTFTPListen  = flag.String("tftp-listen", "0.0.0.0:69", "Address to listen TFTP server")
	flgDHCPListen  = flag.String("dhcp-listen", "0.0.0.0:67", "Address to listen DHCP server")
)

func main() {
	flag.Parse()

	dhcp := &DHCPServer{
		BootFilename: *flgPXEBootFile,
	}
	tftp := &TFTPServer{
		TFTPBootDir: *flgTFTPBootDir,
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		dhcp.Shutdown()
		tftp.Shutdown()
	}()

	var g errgroup.Group

	g.Go(func() error { return dhcp.Start(*flgDHCPListen) })
	g.Go(func() error { return tftp.Start(*flgTFTPListen) })
	err := g.Wait()
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
}
