package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"syscall"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
)

// Mirrors kds-webserver's pkg/udp_discovery.go exactly (same envelope shape,
// same port, same delivery strategy) so kdswebserver's existing UDP listener
// on every terminal picks these up with no changes on that side. Poller and
// webserver are separate binaries with no shared package between them for
// this, hence the duplication - kept intentionally minimal (send-only, no
// listener) since the poller only ever originates one message type.
const udpDiscoveryPort = 2021

type udpEnvelope struct {
	Type   string          `json:"type"`
	Serial string          `json:"serial"`
	Data   json.RawMessage `json:"data"`
}

var udpInstanceID = generateUDPInstanceID()

func generateUDPInstanceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fallback-%d", syscall.Getpid())
	}
	return hex.EncodeToString(buf)
}

// broadcastNewOrder tells every other KDS terminal on the LAN that a brand
// new order was just ingested here, carrying this terminal's own
// OrderFortoday (the "Chk N" number) - see kds-webserver's
// applyNewOrderRemote for what a receiving terminal does with this. Orders
// only ever get created by the poller (this file), never by the webserver's
// own HTTP API in this deployment, so this is the only place a new-order
// broadcast can originate from.
func broadcastNewOrder(db *gorm.DB, orderID int) {
	var order kdsmodels.KDSOrders
	db.Where("id = ?", orderID).First(&order)
	if order.ID == 0 {
		return
	}
	db.Where("kds_order_id = ?", order.ID).Find(&order.Items)
	for i := range order.Items {
		db.Where("kds_item_id = ?", order.Items[i].ID).Find(&order.Items[i].SubItems)
		db.Where("id = ?", order.Items[i].KDSScreenID).Find(&order.Items[i].Screen)
	}

	orderjs, err := json.Marshal(&order)
	if err != nil {
		fmt.Println("udp broadcast: marshal new order failed: ", err)
		return
	}
	envelope := udpEnvelope{Type: "neworderremote", Serial: udpInstanceID, Data: orderjs}
	js, err := json.Marshal(envelope)
	if err != nil {
		fmt.Println("udp broadcast: marshal envelope failed: ", err)
		return
	}

	go func() {
		conn, err := net.ListenPacket("udp4", ":0")
		if err != nil {
			fmt.Println("udp broadcast: socket failed: ", err)
			return
		}
		defer conn.Close()
		if udpConn, ok := conn.(*net.UDPConn); ok {
			if rawConn, err := udpConn.SyscallConn(); err == nil {
				rawConn.Control(func(fd uintptr) {
					syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
				})
			}
		}
		targets := []net.IP{net.IPv4bcast}
		targets = append(targets, localSubnetBroadcastAddresses()...)
		targets = append(targets, localSubnetHostAddresses()...)
		sent := 0
		for _, ip := range targets {
			dest := &net.UDPAddr{IP: ip, Port: udpDiscoveryPort}
			if _, err := conn.WriteTo(js, dest); err != nil {
				continue
			}
			sent++
		}
		fmt.Printf("udp broadcast: new order to %d/%d target(s)\n", sent, len(targets))
	}()
}

// broadcastNewItemsOnOrder tells every other KDS terminal on the LAN that
// itemIDs were just added to an order that already exists on every
// terminal's own copy (see ProcessMessage's existing-order branch in
// engine.go) - mirrors broadcastNewOrder, but for the "item added to an
// already-open ticket" case, which previously had no broadcast at all: a
// terminal that doesn't independently ingest this exact ticket (e.g. a
// Head Chef/monitor terminal, or simply a different prep station whose own
// copy of this order was created by broadcastNewOrder earlier) would only
// ever learn about the order's items as of the moment it was first
// created, never anything added after that.
func broadcastNewItemsOnOrder(db *gorm.DB, orderID int, itemIDs []int) {
	if len(itemIDs) == 0 {
		return
	}
	var order kdsmodels.KDSOrders
	db.Where("id = ?", orderID).First(&order)
	if order.ID == 0 || order.Reference == "" {
		return
	}
	var items []kdsmodels.KDSItems
	db.Where("id IN ?", itemIDs).Find(&items)
	if len(items) == 0 {
		return
	}
	for i := range items {
		db.Where("kds_item_id = ?", items[i].ID).Find(&items[i].SubItems)
		db.Where("id = ?", items[i].KDSScreenID).Find(&items[i].Screen)
	}

	payload := kdsmodels.KDSNewItemsOnOrder{Reference: order.Reference, Items: items}
	payloadjs, err := json.Marshal(&payload)
	if err != nil {
		fmt.Println("udp broadcast: marshal new order items failed: ", err)
		return
	}
	envelope := udpEnvelope{Type: "neworderitemsremote", Serial: udpInstanceID, Data: payloadjs}
	js, err := json.Marshal(envelope)
	if err != nil {
		fmt.Println("udp broadcast: marshal envelope failed: ", err)
		return
	}

	go func() {
		conn, err := net.ListenPacket("udp4", ":0")
		if err != nil {
			fmt.Println("udp broadcast: socket failed: ", err)
			return
		}
		defer conn.Close()
		if udpConn, ok := conn.(*net.UDPConn); ok {
			if rawConn, err := udpConn.SyscallConn(); err == nil {
				rawConn.Control(func(fd uintptr) {
					syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
				})
			}
		}
		targets := []net.IP{net.IPv4bcast}
		targets = append(targets, localSubnetBroadcastAddresses()...)
		targets = append(targets, localSubnetHostAddresses()...)
		sent := 0
		for _, ip := range targets {
			dest := &net.UDPAddr{IP: ip, Port: udpDiscoveryPort}
			if _, err := conn.WriteTo(js, dest); err != nil {
				continue
			}
			sent++
		}
		fmt.Printf("udp broadcast: new order items to %d/%d target(s)\n", sent, len(targets))
	}()
}

// localSubnetBroadcastAddresses and localSubnetHostAddresses are copied
// verbatim from kds-webserver's pkg/udp_discovery.go - see the comments
// there for why the host sweep exists (some networks silently drop both
// 255.255.255.255 and the subnet-directed broadcast address between two
// hosts on the same subnet, while plain unicast between them works fine).
func localSubnetBroadcastAddresses() []net.IP {
	var addrs []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return addrs
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		iaddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range iaddrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			mask := ipnet.Mask
			if ip4 == nil || len(mask) != 4 {
				continue
			}
			bcast := make(net.IP, 4)
			for i := 0; i < 4; i++ {
				bcast[i] = ip4[i] | ^mask[i]
			}
			addrs = append(addrs, bcast)
		}
	}
	return addrs
}

func localSubnetHostAddresses() []net.IP {
	var addrs []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return addrs
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		iaddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range iaddrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			mask := ipnet.Mask
			ones, bits := ipnet.Mask.Size()
			if ip4 == nil || len(mask) != 4 || bits != 32 || ones < 22 {
				continue
			}
			var network [4]byte
			for i := 0; i < 4; i++ {
				network[i] = ip4[i] & mask[i]
			}
			base := binary.BigEndian.Uint32(network[:])
			hostCount := uint32(1) << uint(32-ones)
			self := binary.BigEndian.Uint32(ip4)
			for h := uint32(1); h < hostCount-1; h++ {
				hostVal := base + h
				if hostVal == self {
					continue
				}
				hostIP := make(net.IP, 4)
				binary.BigEndian.PutUint32(hostIP, hostVal)
				addrs = append(addrs, hostIP)
			}
		}
	}
	return addrs
}
