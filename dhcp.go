package main

import (
	"log"
	"net"
	"sync"

	"go.universe.tf/netboot/dhcp4"
)

type DHCPServer struct {
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

		log.Printf("[INFO] Received %s from %s", req.Type, req.HardwareAddr)
		resp := &dhcp4.Packet{
			TransactionID: req.TransactionID,
			HardwareAddr:  req.HardwareAddr,
			ClientAddr:    req.ClientAddr,
			YourAddr:      net.IPv4(172, 24, 32, 1),
			Options:       make(dhcp4.Options),
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
