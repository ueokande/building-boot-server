package main

import (
	"bytes"
	"errors"
	"log"
	"net"
	"sync"

	"go.universe.tf/netboot/dhcp4"
)

type DHCPServer struct {
	BootFilename string

	conn   *dhcp4.Conn
	closed bool
	m      sync.Mutex
}

func (s *DHCPServer) Start(listen string) error {
	var err error
	s.conn, err = dhcp4.NewConn(listen)
	if err != nil {
		log.Fatalf("[FATAL] Unable to listen on %s: %v", listen, err)
	}

	log.Printf("[INFO] Starting DHCP server on %s ...", listen)
	for {
		req, intf, err := s.conn.RecvDHCP()
		if err != nil {
			s.m.Lock()
			if s.closed {
				err = nil
			}
			s.m.Unlock()

			return err
		}
		addr, err := interfaceAddr(intf)
		if err != nil {
			log.Printf("[ERROR] unable to determine an address of %s: %v", intf.Name, err)
			continue
		}

		log.Printf("[INFO] Received %s from %s", req.Type, req.HardwareAddr)
		resp := &dhcp4.Packet{
			TransactionID: req.TransactionID,
			HardwareAddr:  req.HardwareAddr,
			ClientAddr:    req.ClientAddr,
			YourAddr:      net.IPv4(172, 24, 32, 1),
			Options:       make(dhcp4.Options),

			ServerAddr:   addr.IP,
			BootFilename: s.BootFilename,
		}

		resp.Options[dhcp4.OptSubnetMask] = net.IPv4Mask(255, 255, 0, 0)

		switch req.Type {
		case dhcp4.MsgDiscover:
			resp.Type = dhcp4.MsgOffer

		case dhcp4.MsgRequest:
			resp.Type = dhcp4.MsgAck

		default:
			log.Printf("[WARN] message type %s not supported", req.Type)
			continue
		}

		log.Printf("[INFO] Sending %s to %s", resp.Type, resp.HardwareAddr)
		err = s.conn.SendDHCP(resp, intf)
		if err != nil {
			log.Printf("[ERROR] unable to send DHCP packet: %v", err)
		}
	}
	return nil
}

func (s *DHCPServer) Shutdown() error {
	s.m.Lock()
	s.closed = true
	s.m.Unlock()

	return s.conn.Close()
}

// A v4 address has a constant prefix (see https://golang.org/src/net/ip.go?#L58)
var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

func interfaceAddr(intf *net.Interface) (*net.IPNet, error) {
	addrs, err := intf.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && bytes.HasPrefix(ipnet.IP, v4InV6Prefix) {
			return ipnet, nil
		}
	}
	return nil, errors.New("addresses not set")
}
