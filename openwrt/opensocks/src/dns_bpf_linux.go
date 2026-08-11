//go:build linux

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// attachDNSResponseFilter keeps non-DNS LAN traffic in the kernel. Without
// this filter an AF_PACKET socket copies every IPv4 frame into the daemon.
func attachDNSResponseFilter(fd int) error {
	const soAttachFilter = 26
	filters := []syscall.SockFilter{
		{Code: 0x28, K: 12},            // ldh [ether type]
		{Code: 0x15, Jf: 6, K: 0x0800}, // IPv4
		{Code: 0x30, K: 23},            // ldb [IP protocol]
		{Code: 0x15, Jf: 4, K: 17},     // UDP
		{Code: 0xb1, K: 14},            // ldxb 4 * ([IP] & 0xf)
		{Code: 0x48, K: 14},            // ldh [x + UDP source port]
		{Code: 0x15, Jf: 1, K: 53},     // DNS response
		{Code: 0x06, K: 0xffffffff},    // accept
		{Code: 0x06, K: 0},             // reject
	}
	program := syscall.SockFprog{Len: uint16(len(filters)), Filter: &filters[0]}
	_, _, errno := syscall.Syscall6(
		syscall.SYS_SETSOCKOPT,
		uintptr(fd), syscall.SOL_SOCKET, soAttachFilter,
		uintptr(unsafe.Pointer(&program)), unsafe.Sizeof(program), 0,
	)
	if errno != 0 {
		return fmt.Errorf("SO_ATTACH_FILTER: %w", errno)
	}
	return nil
}
