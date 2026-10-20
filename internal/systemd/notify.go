// Package systemd lets the bot talk to the service manager that supervises
// it: it reports when it has finished starting, and keeps a watchdog fed so
// a wedged process is noticed and replaced rather than sitting there
// "active" and answering nobody.
//
// Every function is a no-op when the process is not running under systemd,
// so the bot still runs from a terminal exactly as before.
package systemd

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// notifySocket returns the address systemd is listening on, or "" when this
// process has no service manager.
func notifySocket() string {
	socket := os.Getenv("NOTIFY_SOCKET")
	if socket == "" {
		return ""
	}

	// A leading '@' means an abstract socket, whose real name starts with a
	// NUL byte.
	if strings.HasPrefix(socket, "@") {
		return "\x00" + socket[1:]
	}

	return socket
}

// notify sends one state line to the service manager.
func notify(state string) error {
	socket := notifySocket()
	if socket == "" {
		return nil
	}

	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		return fmt.Errorf("failed to reach the service manager: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(state)); err != nil {
		return fmt.Errorf("failed to notify the service manager: %w", err)
	}

	return nil
}

// Ready tells systemd the bot has finished starting and is serving.
//
// The unit is Type=notify, so until this is sent systemd considers the
// service to still be starting — which is what makes a bot that cannot
// reach Telegram at boot count as a failed start rather than a healthy one.
func Ready() error {
	return notify("READY=1\nSTATUS=Listening for messages")
}

// Stopping tells systemd a clean shutdown is under way.
func Stopping() error {
	return notify("STOPPING=1\nSTATUS=Shutting down")
}

// Status updates the one-line description shown by `systemctl status`.
func Status(text string) error {
	return notify("STATUS=" + text)
}

// WatchdogInterval returns how often the watchdog must be fed, or zero when
// no watchdog is configured.
//
// systemd reports the timeout; feeding it at half that leaves room for one
// missed tick before the process is killed.
func WatchdogInterval() time.Duration {
	raw := os.Getenv("WATCHDOG_USEC")
	if raw == "" {
		return 0
	}

	// WATCHDOG_PID, when set, names the only process meant to feed it.
	if pid := os.Getenv("WATCHDOG_PID"); pid != "" {
		if pid != strconv.Itoa(os.Getpid()) {
			return 0
		}
	}

	usec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || usec <= 0 {
		return 0
	}

	return time.Duration(usec) * time.Microsecond / 2
}

// StartWatchdog feeds the systemd watchdog until stop is closed.
//
// It detects a process that is still alive but no longer doing anything —
// a deadlock, or a runtime that has stopped scheduling — which a plain
// Restart=always cannot see, because from the outside such a process looks
// perfectly healthy.
//
// It deliberately does NOT check that Telegram is reachable. The link to
// api.telegram.org from this server is not always available, and tying the
// watchdog to it would turn a network outage into a restart loop, which
// fixes nothing and buries the real cause in restart noise.
func StartWatchdog(stop <-chan struct{}) (interval time.Duration, running bool) {
	interval = WatchdogInterval()
	if interval <= 0 {
		return 0, false
	}

	var once sync.Once

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := notify("WATCHDOG=1"); err != nil {
					// Log once rather than every tick; a broken socket is
					// not something retrying will fix.
					once.Do(func() {
						fmt.Fprintf(os.Stderr, "watchdog: %v\n", err)
					})
				}
			case <-stop:
				return
			}
		}
	}()

	return interval, true
}
