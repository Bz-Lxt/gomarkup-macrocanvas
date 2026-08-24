//go:build !linux && !darwin && !windows

package main

import "fmt"

func runAgent(backend, token string) error {
	return fmt.Errorf("mc-agent unsupported on this OS (backend=%s token_set=%t)", backend, token != "")
}
