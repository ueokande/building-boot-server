package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.universe.tf/netboot/tftp"
)

type TFTPServer struct {
	PXEPathPrefix    string
	KernelPathPrefix string
	IPXERomPath      string

	conn   net.PacketConn
	closed bool
	m      sync.Mutex
}

func (s *TFTPServer) Start(listen string) error {
	srv := &tftp.Server{Handler: s.handle}

	log.Printf("[INFO] Starting TFTP server on %s ...", listen)

	var err error
	s.conn, err = net.ListenPacket("udp4", listen)
	if err != nil {
		return err
	}
	err = srv.Serve(s.conn)
	if err != nil {
		s.m.Lock()
		if s.closed {
			err = nil
		}
		s.m.Unlock()
	}
	return err
}

func (s *TFTPServer) Shutdown() error {
	s.m.Lock()
	s.closed = true
	s.m.Unlock()

	return s.conn.Close()
}

func (s *TFTPServer) handle(path string, addr net.Addr) (io.ReadCloser, int64, error) {
	log.Printf("[INFO] GET %s from %s", path, addr)
	switch {
	case path == "pxelinux/pxelinux.cfg/default":
		return s.handlePXEConfig()
	case strings.HasPrefix(path, "pxelinux/boot/"):
		return s.handleKernelImages(strings.TrimPrefix(path, "pxelinux/boot/"))
	case strings.HasPrefix(path, "pxelinux/"):
		return s.handlePXEImages(strings.TrimPrefix(path, "pxelinux/"))
	case path == "undionly.kpxe":
		return s.handleIPXEImage(path)
	}
	return nil, 0, errors.New("not found")
}

func (s *TFTPServer) handlePXEConfig() (io.ReadCloser, int64, error) {
	const cfg = `default linux

label linux
  kernel boot/vmlinuz
  append initrd=boot/initrd.img console=ttyS0
`
	r := ioutil.NopCloser(strings.NewReader(cfg))
	return r, int64(len(cfg)), nil
}

func (s *TFTPServer) handlePXEImages(path string) (io.ReadCloser, int64, error) {
	path = filepath.Join(s.PXEPathPrefix, path)
	return s.handleFile(path)
}

func (s *TFTPServer) handleKernelImages(path string) (io.ReadCloser, int64, error) {
	path = filepath.Join(s.KernelPathPrefix, path)
	return s.handleFile(path)
}

func (s *TFTPServer) handleIPXEImage(path string) (io.ReadCloser, int64, error) {
	return s.handleFile(s.IPXERomPath)
}

func (s *TFTPServer) handleFile(path string) (io.ReadCloser, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		return nil, 0, err
	}
	fi, err := f.Stat()
	if err != nil {
		log.Printf("[ERROR] %v", err)
		return nil, 0, err
	}
	return f, fi.Size(), err
}
