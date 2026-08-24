//go:build windows

package main

/*
#include <windows.h>

static HHOOK hook = NULL;

static LRESULT CALLBACK KeyboardProc(int nCode, WPARAM wParam, LPARAM lParam) {
    return CallNextHookEx(hook, nCode, wParam, lParam);
}

int mc_hook_install(void) {
    hook = SetWindowsHookExW(WH_KEYBOARD_LL, KeyboardProc, NULL, 0);
    return hook == NULL ? 1 : 0;
}

void mc_hook_remove(void) {
    if (hook) {
        UnhookWindowsHookEx(hook);
        hook = NULL;
    }
}
*/
import "C"

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func runAgent(backend, token string) error {
	if C.mc_hook_install() != 0 {
		return fmt.Errorf("SetWindowsHookEx failed (backend=%s)", backend)
	}
	defer C.mc_hook_remove()
	fmt.Fprintf(os.Stderr, "mc-agent: SetWindowsHookEx installed, would relay to %s (token_len=%d). V2 relay loop not enabled.\n", backend, len(token))
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	return nil
}
