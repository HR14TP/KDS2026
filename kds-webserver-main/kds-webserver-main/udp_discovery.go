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
)

// UDP discovery lets every KDS terminal on the local network pick up
// item/order status changes from any other KDS terminal automatically - no
// screen1/screen2/peers IP address has to be configured. Every status change
// is broadcast on the local subnet; every terminal listens for those
// broadcasts and applies the same match-by-Reference logic the existing
// "duplication" remote handlers already use - if the referenced order/item
// isn't on this terminal's own copy of the order, the broadcast is silently
// ignored, exactly like it already is for the IP-based paths.
const udpDiscoveryPort = 2021

type udpEnvelope struct {
	Type   string          `json:"type"`
	Serial string          `json:"serial"`
	Data   json.RawMessage `json:"data"`
}

// udpInstanceID tags our own broadcasts so we can recognize and ignore our
// own echo. This is deliberately NOT ENV.Config.Client.Serial - that field
// is a cloud-registration code that's blank/identical-default on terminals
// that were never individually registered, which is the norm for KDS units
// on a closed LAN. Two terminals sharing the same Client.Serial would each
// mistake every one of the OTHER terminal's broadcasts for their own echo
// and silently discard them, breaking sync in both directions. A random
// value generated fresh per process is guaranteed unique regardless of
// registration/config state.
var udpInstanceID = generateUDPInstanceID()

func generateUDPInstanceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// Vanishingly unlikely, but fall back to something still unique
		// enough per-process rather than leaving it empty.
		return fmt.Sprintf("fallback-%d", syscall.Getpid())
	}
	return hex.EncodeToString(buf)
}

// startUDPDiscovery opens the broadcast listener once. Safe to call more
// than once - later calls are ignored once the listener is up.
var udpListenerStarted = false

func startUDPDiscovery() {
	if udpListenerStarted {
		return
	}
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: udpDiscoveryPort}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		fmt.Println("udp discovery: listen failed: ", err)
		return
	}
	udpListenerStarted = true
	fmt.Println("udp discovery: listening on", addr.String())
	bcastAddrs := localSubnetBroadcastAddresses()
	if len(bcastAddrs) == 0 {
		fmt.Println("udp discovery: no local subnet broadcast addresses found - only the global 255.255.255.255 address will be used")
	} else {
		for _, ip := range bcastAddrs {
			fmt.Println("udp discovery: will broadcast to", ip.String())
		}
	}
	go func() {
		buf := make([]byte, 65535)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Println("udp discovery: read error: ", err)
				continue
			}
			handleUDPEnvelope(buf[:n], addr.IP.String())
		}
	}()
}

func handleUDPEnvelope(raw []byte, senderIP string) {
	var envelope udpEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return
	}
	// Ignore our own broadcast - broadcast sockets bound to 0.0.0.0 commonly
	// receive a copy of what they just sent.
	if envelope.Serial == udpInstanceID {
		return
	}
	switch envelope.Type {
	case "closeitemremote":
		var item kdsmodels.KDSItems
		if err := json.Unmarshal(envelope.Data, &item); err == nil {
			applyCloseRemoteItem(item)
			notifyKDSOrders()
		}
	case "openitemremote":
		var item kdsmodels.KDSItems
		if err := json.Unmarshal(envelope.Data, &item); err == nil {
			applyOpenRemoteItem(item)
			notifyKDSOrders()
		}
	case "startproductionremote":
		var item kdsmodels.KDSItems
		if err := json.Unmarshal(envelope.Data, &item); err == nil {
			applyStartProductionRemote(item)
			notifyKDSOrders()
		}
	case "clearproductionremote":
		var item kdsmodels.KDSItems
		if err := json.Unmarshal(envelope.Data, &item); err == nil {
			applyClearProductionRemote(item)
			notifyKDSOrders()
		}
	case "sendtotableremote":
		var item kdsmodels.KDSItems
		if err := json.Unmarshal(envelope.Data, &item); err == nil {
			applySendToTableRemote(item)
			notifyKDSOrders()
		}
	case "neworderremote":
		var order kdsmodels.KDSOrders
		if err := json.Unmarshal(envelope.Data, &order); err == nil {
			applyNewOrderRemote(order)
			notifyKDSOrders()
		}
	case "neworderitemsremote":
		var payload kdsmodels.KDSNewItemsOnOrder
		if err := json.Unmarshal(envelope.Data, &payload); err == nil {
			applyNewOrderItemsRemote(payload)
			notifyKDSOrders()
		}
	case "closeorderremote":
		var order kdsmodels.KDSOrders
		if err := json.Unmarshal(envelope.Data, &order); err == nil {
			applyCloseOrderRemote(order)
		}
	case "closeorderscreenremote":
		var closemsg kdsmodels.KDSOrderScreenClose
		if err := json.Unmarshal(envelope.Data, &closemsg); err == nil {
			applyCloseOrderScreenRemote(closemsg)
		}
	case "recallorderscreenremote":
		var recallmsg kdsmodels.KDSOrderScreenRecall
		if err := json.Unmarshal(envelope.Data, &recallmsg); err == nil {
			applyRecallOrderScreenRemote(recallmsg)
		}
	case "screenpassassignmentremote":
		var assignment kdsmodels.KDSScreenPassAssignment
		if err := json.Unmarshal(envelope.Data, &assignment); err == nil {
			upsertScreenPassAssignment(assignment.Screen, assignment.Passterminal)
			upsertKnownTerminal(assignment.Screen, senderIP)
		}
	case "passterminalannounceremote":
		var announce kdsmodels.KDSPassTerminalAnnounce
		if err := json.Unmarshal(envelope.Data, &announce); err == nil {
			upsertPassTerminalAnnounce(announce.Name)
		}
	case "discoverpingremote":
		// UDP leg of a "Rediscover Pass Terminals" request (see
		// requestPassTerminalRediscovery in pass_terminal_assignment.go) -
		// no payload to decode, receiving this at all is the signal to
		// re-announce ourselves right now rather than waiting for the next
		// periodic 2-minute tick.
		asyncAnnouncePassTerminalInfo()
	}
}

// localSubnetBroadcastAddresses returns the subnet-directed broadcast
// address (e.g. 192.168.1.255) of every "real" IPv4 interface on this
// machine. Relying on the single global address (255.255.255.255) sends the
// packet out whatever interface the OS routing table treats as default only
// - on a KDS PC with more than one active NIC (wired + wifi is common), or
// on networks/switches that only forward subnet-directed broadcasts, that
// silently misses terminals reachable through the other interface. Sending
// to every interface's own broadcast address as well covers both cases.
func localSubnetBroadcastAddresses() []net.IP {
	var addrs []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("udp discovery: could not list interfaces: ", err)
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

// localSubnetHostAddresses enumerates every individual host address on each
// local interface's own subnet (excluding the network and broadcast
// addresses, and this machine's own address). This exists because some
// networks - confirmed in the field: same subnet, same switch, no firewall
// on either machine - silently drop both 255.255.255.255 and the
// subnet-directed broadcast address between two hosts, while a perfectly
// ordinary unicast packet between those same two hosts goes through with no
// issue. Sweeping every host address as plain unicast sidesteps whatever is
// eating the broadcast, at the cost of a handful more small UDP sends per
// event - negligible for how infrequently these fire (an order changing
// status), and still zero-config since it's derived from the interface's
// own subnet, not typed in anywhere. Capped to /22 or smaller (at most 1022
// addresses) so this can't turn into a flood on an unexpectedly large
// network - broadcastUDP's other two delivery paths still cover that case.
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

// broadcastUDP fans a change out to every KDS terminal on the local
// network(s) this machine is on, tagged with our own serial so a broadcast
// that's delivered back to ourselves gets ignored instead of double-applying
// the change. Sends to the global broadcast address, every local
// interface's own subnet-directed broadcast address, and (since some
// networks silently drop both of those between hosts - see
// localSubnetHostAddresses) every individual host address on those subnets
// too. Every caller is a synchronous HTTP handler that a KDS terminal is
// waiting on (e.g. clicking START on an item), so the actual network sends
// - up to a couple hundred of them once the host sweep is included - run in
// a goroutine instead of blocking the response on every single one
// completing first.
func broadcastUDP(msgType string, data []byte) {
	envelope := udpEnvelope{Type: msgType, Serial: udpInstanceID, Data: data}
	js, err := json.Marshal(envelope)
	if err != nil {
		fmt.Println(err)
		return
	}
	go func() {
		conn, err := net.ListenPacket("udp4", ":0")
		if err != nil {
			fmt.Println("udp discovery: broadcast socket failed: ", err)
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
				fmt.Println("udp discovery: broadcast send to "+ip.String()+" failed: ", err)
				continue
			}
			sent++
		}
		fmt.Printf("udp discovery: broadcast %s to %d/%d target(s)\n", msgType, sent, len(targets))
	}()
}
