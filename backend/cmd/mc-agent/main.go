package main

import (
	"flag"
	"fmt"
	"os"
)

// mc-agent is the V2 host-native hook process (T-N).
// Docker image does not ship this binary. Build on the host:
//   go build -o mc-agent ./cmd/mc-agent
func main() {
	backend := flag.String("backend", "ws://127.0.0.1:31822", "backend websocket base")
	token := flag.String("token", "mc-dev-31821", "auth token")
	flag.Parse()
	if err := runAgent(*backend, *token); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
