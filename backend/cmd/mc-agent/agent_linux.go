//go:build linux

package main

import "fmt"

func runAgent(backend, token string) error {
	return fmt.Errorf("mc-agent on linux is unused: the container already owns evdev/uinput (backend=%s token_set=%t)", backend, token != "")
}
