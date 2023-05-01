//go:build openbsd
// +build openbsd

package protector

import (
	"log"

	"golang.org/x/sys/unix"
)

// Source https://github.com/junegunn/fzf/blob/a1bcdc225e1c9b890214fcea3d19d85226fc552a/src/protector/protector_openbsd.go

// Protect calls OS specific protections like pledge on OpenBSD
func Protect(writePath string) {
	mandatoryDNSfiles := []string{
		"/etc/resolv.conf",
	}
	for _, fname := range mandatoryDNSfiles {
		if err := unix.Unveil(fname, "r"); err != nil {
			log.Panicf("Error unveiling: %s", err)
		}
	}
	if err := unix.Unveil(writePath, "w"); err != nil {
		log.Panicf("Error unveiling: %s", err)
	}
	unix.UnveilBlock()
	err := unix.PledgePromises("stdio cpath rpath wpath tty inet dns")
	if err != nil {
		log.Panicf("Error in pledge: %s", err)
	}
}
