package main

import (
	"log"
	"net"

	"go.universe.tf/netboot/dhcp4"
)

func main() {
	listen := "0.0.0.0:67"
	conn, err := dhcp4.NewConn(listen)
	if err != nil {
		log.Fatalf("[FATAL] Unable to listen on %s: %v", listen, err)
	}
	defer conn.Close()

	log.Printf("[INFO] Starting DHCP server...")
	for {
		req, intf, err := conn.RecvDHCP()
		if err != nil {
			log.Fatalf("[ERROR] Failed to receive DHCP package: %v", err)
		}

		log.Printf("[INFO] Received %s from %s", req.Type, req.HardwareAddr)
		resp := &dhcp4.Packet{
			TransactionID: req.TransactionID,
			HardwareAddr:  req.HardwareAddr,
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
		err = conn.SendDHCP(resp, intf)
		if err != nil {
			log.Printf("[ERROR] unable to send DHCP packet: %v", err)
		}
	}
}
