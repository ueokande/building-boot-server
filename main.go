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

	g.Go(func() error { return dhcp.Start("0.0.0.0:67") })
	g.Go(func() error { return tftp.Start("0.0.0.0:69") })
	err := g.Wait()
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
}
