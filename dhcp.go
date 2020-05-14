package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"go.universe.tf/netboot/dhcp4"
)

type VendorClass int

const (
	PXEClientBIOS VendorClass = iota
	PXEClientX86
	PXEClientX64
	HTTPClientX86
	HTTPClientX64
)

func (v VendorClass) String() string {
	switch v {
	case PXEClientBIOS:
		return "PXEClient (BIOS)"
	case PXEClientX86:
		return "PXEClient (x86)"
	case PXEClientX64:
		return "PXEClient (x64)"
	case HTTPClientX86:
		return "HTTPClient (x86)"
	case HTTPClientX64:
		return "HTTPClient (x64)"
	}
	panic("unexpected vendor class")
}

type unknownVendorClassError struct {
	VendorClass string
}

func (e *unknownVendorClassError) Error() string {
	return fmt.Sprintf("unknown vendor class %q", e.VendorClass)
}

var errVendorClassNotPresent = errors.New("vendor-class identifier not presented")

const OptUserClassIdentification = 77

type DHCPServer struct {
	TFTPBootFile     string
	IPXEHTTPBootFile string

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

		addr, err := interfaceAddr(intf)
		if err != nil {
			log.Printf("[ERROR] unable to determine an address of %s: %v", intf.Name, err)
			continue
		}
		vendorclass, err := detectVendorClass(req)
		if err == errVendorClassNotPresent {
			log.Printf("[WARN] Vendor-Class not presented")
			continue
		} else if err != nil {
			if err, ok := err.(*unknownVendorClassError); ok {
				log.Printf("[WARN] Unsupported Vendor-Classs: %s", err.VendorClass)
			} else {
			}
			log.Printf("[WARN] Unable to get Vendor class identifier: %v", err)
			continue
		}

		resp := &dhcp4.Packet{
			TransactionID: req.TransactionID,
			HardwareAddr:  req.HardwareAddr,
			ClientAddr:    req.ClientAddr,
			YourAddr:      net.IPv4(172, 24, 32, 1),
			Options:       make(dhcp4.Options),
			ServerAddr:    addr.IP,
		}

		resp.Options[dhcp4.OptSubnetMask] = addr.Mask
		resp.Options[dhcp4.OptServerIdentifier] = addr.IP.To4()

		switch vendorclass {
		case PXEClientBIOS:
			userclass, err := req.Options.String(OptUserClassIdentification)
			if err == nil && userclass == "iPXE" {
				s.handleIPXEBoot(resp)
			} else {
				s.handlePXEBoot(resp)
			}
		default:
			log.Printf("[WARN] Unsupported vendorclass %q", vendorclass)
			continue
		}

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

func detectVendorClass(req *dhcp4.Packet) (VendorClass, error) {
	_, ok := req.Options[dhcp4.OptVendorIdentifier]
	if !ok {
		return 0, errVendorClassNotPresent
	}
	vendorclass, err := req.Options.String(dhcp4.OptVendorIdentifier)
	if err != nil {
		return 0, err
	}
	if strings.HasPrefix(vendorclass, "PXEClient:Arch:00000:") {
		return PXEClientBIOS, nil
	} else if strings.HasPrefix(vendorclass, "PXEClient:Arch:00006:") {
		return PXEClientX86, nil
	} else if strings.HasPrefix(vendorclass, "PXEClient:Arch:00007:") {
		return PXEClientX64, nil
	} else if strings.HasPrefix(vendorclass, "HTTPClient:Arch:00015:") {
		return HTTPClientX86, nil
	} else if strings.HasPrefix(vendorclass, "HTTPClient:Arch:00016:") {
		return HTTPClientX64, nil
	}
	return -1, &unknownVendorClassError{VendorClass: vendorclass}
}

func (s *DHCPServer) handleIPXEBoot(pkt *dhcp4.Packet) {
	pkt.BootFilename = fmt.Sprintf("http://%s/%s", pkt.ServerAddr, s.IPXEHTTPBootFile)
}

func (s *DHCPServer) handlePXEBoot(pkt *dhcp4.Packet) {
	pkt.BootFilename = s.TFTPBootFile
}
