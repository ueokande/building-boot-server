package main

import (
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"go.universe.tf/netboot/tftp"
)

type TFTPServer struct {
	TFTPDir string

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

	f, err := os.Open(filepath.Join(s.TFTPDir, path))
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
