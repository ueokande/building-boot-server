package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"golang.org/x/sync/errgroup"
)

var (
	flgPXEBIOSBootFile  = flag.String("pxe-bios-boot-file", "pxelinux/pxelinux.0", "The file name used in PXE (Legacy BIOS)")
	flgIPXEBIOSBootFile = flag.String("ipxe-bios-boot-file", "boot.ipxe", "The file name used in iPXE (Legacy BIOS)")

	flgTFTPDir = flag.String("tftp-dir", "./tftpboot", "The base directory including files served by TFTP server")
	flgHTTPDir = flag.String("http-dir", "./httpboot", "The base directory including files served by HTTP server")
)

func main() {
	flag.Parse()

	dhcp := &DHCPServer{
		PXEBIOSBootFile:  *flgPXEBIOSBootFile,
		IPXEBIOSBootFile: *flgIPXEBIOSBootFile,
	}
	tftp := &TFTPServer{
		TFTPDir: *flgTFTPDir,
	}
	http := &HTTPServer{
		HTTPDir: *flgHTTPDir,
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		dhcp.Shutdown()
		tftp.Shutdown()
		http.Shutdown()
	}()

	var g errgroup.Group

	g.Go(func() error { return dhcp.Start("0.0.0.0:67") })
	g.Go(func() error { return tftp.Start("0.0.0.0:69") })
	g.Go(func() error { return http.Start("0.0.0.0:80") })
	err := g.Wait()
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
}
