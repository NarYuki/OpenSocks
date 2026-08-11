//go:build !linux

package main

import "fmt"

func attachDNSResponseFilter(int) error {
	return fmt.Errorf("packet filter is supported only on Linux")
}
