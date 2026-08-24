package timing

import (
	"os"
	"runtime"
	"strings"
)

// RTAvailable reports whether enabling SCHED_FIFO is safe.
// cgroup v2 has no RT bandwidth file → treat as unsafe (feasibility REPORT B-4).
func RTAvailable() (ok bool, reason string) {
	if runtime.GOOS != "linux" {
		return false, "not linux"
	}
	if b, err := os.ReadFile("/sys/fs/cgroup/cpu.rt_runtime_us"); err == nil {
		v := strings.TrimSpace(string(b))
		if v == "" || v == "0" {
			return false, "cpu.rt_runtime_us=0"
		}
		return true, "cgroup v1 rt budget " + v
	}
	if _, err := os.Stat("/sys/fs/cgroup/cpu.max"); err == nil {
		return false, "cgroup v2 has no RT bandwidth; SCHED_FIFO would livelock"
	}
	return false, "rt interface absent"
}
