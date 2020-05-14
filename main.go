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

	flgDHCPListen = flag.String("dhcp-listen", "0.0.0.0:67", "Address and port to listen for DHCP requests on")
	flgTFTPListen = flag.String("tftp-listen", "0.0.0.0:69", "Address and port to listen for TFTP requests on")
	flgHTTPListen = flag.String("http-listen", "0.0.0.0:80", "Address and port to listen for HTTP requests on")
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

	g.Go(func() error { return dhcp.Start(*flgDHCPListen) })
	g.Go(func() error { return tftp.Start(*flgTFTPListen) })
	g.Go(func() error { return http.Start(*flgHTTPListen) })
	err := g.Wait()
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
}
