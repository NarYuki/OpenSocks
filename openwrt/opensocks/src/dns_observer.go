package main

import (
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"syscall"
	"time"
)

// observeDNSAnswers passively learns addresses for every subdomain of the
// configured service suffixes. It does not sit in the DNS path, so a failure
// here can never break client name resolution.
func observeDNSAnswers() {
	const (
		afPacket       = 17
		ethPIP         = 0x0800
		soBindToDevice = 25
	)
	device := detectLANDevice()
	trustedIP, trustedMAC := lanIdentity(device)
	if trustedIP == nil || len(trustedMAC) != 6 {
		logf("dns observer unavailable: LAN identity could not be verified")
		return
	}
	fd, err := syscall.Socket(afPacket, syscall.SOCK_RAW, int(htons(ethPIP)))
	if err != nil {
		logf("dns observer unavailable: %v", err)
		return
	}
	defer syscall.Close(fd)
	_ = syscall.SetsockoptString(fd, syscall.SOL_SOCKET, soBindToDevice, device)
	if err := attachDNSResponseFilter(fd); err != nil {
		logf("dns observer disabled because the kernel filter could not be attached: %v", err)
		return
	}

	type learning struct {
		group string
		ips   []string
	}
	learned := make(chan learning, 128)
	go func() {
		pending := map[string]map[string]bool{}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case item := <-learned:
				if pending[item.group] == nil {
					pending[item.group] = map[string]bool{}
				}
				for _, ip := range item.ips {
					pending[item.group][ip] = true
				}
			case <-ticker.C:
				flushLearnedDNS(pending)
				pending = map[string]map[string]bool{}
			}
		}
	}()
	buf := make([]byte, 8192)
	for {
		n, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil || n < 42 {
			continue
		}
		names, ips := dnsIPv4Answers(buf[:n], trustedIP, trustedMAC)
		if len(ips) > 0 {
			if chinaRouteForDNSNames(names) {
				select {
				case learned <- learning{group: "", ips: ips}:
				default:
				}
			}
			for _, group := range serviceGroupsForDNSNames(names) {
				select {
				case learned <- learning{group: group, ips: ips}:
				default:
				}
			}
		}
	}
}

func lanIdentity(device string) (net.IP, net.HardwareAddr) {
	ifc, err := net.InterfaceByName(device)
	if err != nil {
		return nil, nil
	}
	addrs, err := ifc.Addrs()
	if err != nil {
		return nil, nil
	}
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err == nil && ip.To4() != nil {
			return ip.To4(), ifc.HardwareAddr
		}
	}
	return nil, nil
}

func htons(v uint16) uint16 { return v<<8 | v>>8 }

func serviceForDNSNames(names []string) string {
	groups := serviceGroupsForDNSNames(names)
	if len(groups) > 0 {
		return groups[0]
	}
	return ""
}

func chinaRouteForDNSNames(names []string) bool {
	for _, name := range names {
		n := strings.TrimSuffix(strings.ToLower(name), ".")
		if n == "cn" || strings.HasSuffix(n, ".cn") {
			return true
		}
		for _, suffix := range cnDomainSuffixes {
			s := strings.TrimPrefix(strings.ToLower(suffix), ".")
			if n == s || strings.HasSuffix(n, "."+s) {
				return true
			}
		}
	}
	return false
}

// A DNS response can contain a service hostname followed by one or more CDN
// CNAMEs. Keep every matching service association; nftables applies the stable
// group priority and counts the flow only once.
func serviceGroupsForDNSNames(names []string) []string {
	matched := make([]string, 0, 2)
	for _, g := range chinaServiceGroups { // specific groups retain priority
		found := false
		for _, name := range names {
			for _, suffix := range g.Domains {
				n, s := strings.TrimSuffix(strings.ToLower(name), "."), strings.TrimPrefix(strings.ToLower(suffix), ".")
				if n == s || strings.HasSuffix(n, "."+s) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			matched = append(matched, g.Name)
		}
	}
	return matched
}

func flushLearnedDNS(pending map[string]map[string]bool) {
	sets := map[string][]string{}
	for group, values := range pending {
		if len(values) == 0 {
			continue
		}
		ips := make([]string, 0, len(values))
		for ip := range values {
			ips = append(ips, ip)
		}
		sets["domain4"] = append(sets["domain4"], ips...)
		if group != "" {
			sets["svc_"+group+"4"] = append(sets["svc_"+group+"4"], ips...)
		}
	}
	addDynamicElements(sets)
}

func dnsIPv4Answers(frame []byte, _ net.IP, trustedMAC net.HardwareAddr) ([]string, []string) {
	if len(frame) < 34 || binary.BigEndian.Uint16(frame[12:14]) != 0x0800 {
		return nil, nil
	}
	if len(trustedMAC) != 6 || !bytes.Equal(frame[6:12], trustedMAC) {
		return nil, nil
	}
	ip := 14
	ihl := int(frame[ip]&0x0f) * 4
	if ihl < 20 || len(frame) < ip+ihl+8 || frame[ip+9] != 17 {
		return nil, nil
	}
	// A DNS request transparently redirected from an external resolver is
	// returned to the LAN with that resolver's source IP restored by conntrack.
	// The router's verified source MAC is therefore the stable trust boundary;
	// requiring its LAN IP here would discard exactly those captured replies.
	udp := ip + ihl
	if binary.BigEndian.Uint16(frame[udp:udp+2]) != 53 {
		return nil, nil
	}
	dns := frame[udp+8:]
	if len(dns) < 12 || dns[2]&0x80 == 0 {
		return nil, nil
	}
	qd, an := int(binary.BigEndian.Uint16(dns[4:6])), int(binary.BigEndian.Uint16(dns[6:8]))
	off, names := 12, []string{}
	for i := 0; i < qd; i++ {
		name, next, ok := dnsName(dns, off, 0)
		if !ok || next+4 > len(dns) {
			return names, nil
		}
		names, off = append(names, name), next+4
	}
	ips := []string{}
	for i := 0; i < an && off < len(dns); i++ {
		name, next, ok := dnsName(dns, off, 0)
		if !ok || next+10 > len(dns) {
			break
		}
		typ, rdlen := binary.BigEndian.Uint16(dns[next:next+2]), int(binary.BigEndian.Uint16(dns[next+8:next+10]))
		rdata := next + 10
		if rdata+rdlen > len(dns) {
			break
		}
		names = append(names, name)
		if typ == 1 && rdlen == 4 {
			ips = append(ips, netIPv4String(dns[rdata:rdata+4]))
		} else if typ == 5 {
			if cname, _, ok := dnsName(dns, rdata, 0); ok {
				names = append(names, cname)
			}
		}
		off = rdata + rdlen
	}
	return names, ips
}

func dnsName(msg []byte, off, depth int) (string, int, bool) {
	if depth > 8 || off >= len(msg) {
		return "", off, false
	}
	parts, next := []string{}, off
	for off < len(msg) {
		l := int(msg[off])
		if l&0xc0 == 0xc0 {
			if off+1 >= len(msg) {
				return "", next, false
			}
			ptr := (l&0x3f)<<8 | int(msg[off+1])
			s, _, ok := dnsName(msg, ptr, depth+1)
			if !ok {
				return "", next, false
			}
			parts = append(parts, s)
			return strings.Join(parts, "."), off + 2, true
		}
		off++
		if l == 0 {
			return strings.Join(parts, "."), off, true
		}
		if l > 63 || off+l > len(msg) {
			return "", next, false
		}
		parts = append(parts, string(msg[off:off+l]))
		off += l
	}
	return "", next, false
}

func netIPv4String(b []byte) string {
	return strings.Join([]string{itoaByte(b[0]), itoaByte(b[1]), itoaByte(b[2]), itoaByte(b[3])}, ".")
}

func itoaByte(v byte) string {
	if v >= 100 {
		return string([]byte{'0' + v/100, '0' + (v/10)%10, '0' + v%10})
	}
	if v >= 10 {
		return string([]byte{'0' + v/10, '0' + v%10})
	}
	return string([]byte{'0' + v})
}
