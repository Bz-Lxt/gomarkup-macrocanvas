//go:build darwin

package main

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics
#include <ApplicationServices/ApplicationServices.h>
#include <CoreGraphics/CoreGraphics.h>

static CFMachPortRef tap = NULL;

static CGEventRef tapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
    return event;
}

int mc_tap_create(void) {
    tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionListenOnly,
        CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp) | CGEventMaskBit(kCGEventFlagsChanged),
        tapCallback, NULL);
    return tap == NULL ? 1 : 0;
}

void mc_tap_release(void) {
    if (tap) {
        CFRelease(tap);
        tap = NULL;
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
	if C.mc_tap_create() != 0 {
		return fmt.Errorf("CGEventTapCreate failed: grant Input Monitoring / Accessibility, then retry (backend=%s)", backend)
	}
	defer C.mc_tap_release()
	fmt.Fprintf(os.Stderr, "mc-agent: CGEventTap installed, would relay to %s (token_len=%d). V2 relay loop not enabled in this build.\n", backend, len(token))
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	return nil
}
