package systemd

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// listen stands in for systemd: a datagram socket that records what the bot
// sends it.
func listen(t *testing.T) (addr string, received <-chan string) {
	t.Helper()

	// Unix socket paths are capped at about 104 bytes, which the long
	// directory names t.TempDir() builds from test names can exceed.
	dir, err := os.MkdirTemp("", "sd")
	if err != nil {
		t.Fatalf("failed to make a temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	path := filepath.Join(dir, "n.sock")

	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Fatalf("failed to open the stub socket: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	messages := make(chan string, 16)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				close(messages)
				return
			}
			messages <- string(buf[:n])
		}
	}()

	return path, messages
}

// TestNotifyIsSilentOutsideSystemd: the bot must still run from a terminal.
func TestNotifyIsSilentOutsideSystemd(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")

	if err := Ready(); err != nil {
		t.Fatalf("Ready outside systemd returned an error: %v", err)
	}

	if err := Status("anything"); err != nil {
		t.Fatalf("Status outside systemd returned an error: %v", err)
	}
}

func TestReadyReachesTheServiceManager(t *testing.T) {
	addr, messages := listen(t)
	t.Setenv("NOTIFY_SOCKET", addr)

	if err := Ready(); err != nil {
		t.Fatalf("Ready returned an error: %v", err)
	}

	select {
	case got := <-messages:
		if got == "" || got[:8] != "READY=1\n" {
			t.Fatalf("sent %q, want it to start with READY=1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nothing reached the service manager")
	}
}

// TestWatchdogIntervalIsHalfTheTimeout leaves room for one missed tick.
func TestWatchdogIntervalIsHalfTheTimeout(t *testing.T) {
	t.Setenv("WATCHDOG_USEC", "120000000") // 120s
	t.Setenv("WATCHDOG_PID", "")

	if got := WatchdogInterval(); got != 60*time.Second {
		t.Fatalf("interval = %s, want 60s", got)
	}
}

func TestWatchdogIntervalIsZeroWhenUnconfigured(t *testing.T) {
	t.Setenv("WATCHDOG_USEC", "")

	if got := WatchdogInterval(); got != 0 {
		t.Fatalf("interval = %s, want 0", got)
	}

	// Garbage must not be read as a valid interval either.
	for _, bad := range []string{"abc", "0", "-5"} {
		t.Setenv("WATCHDOG_USEC", bad)

		if got := WatchdogInterval(); got != 0 {
			t.Fatalf("WATCHDOG_USEC=%q gave %s, want 0", bad, got)
		}
	}
}

// TestWatchdogIgnoresAnotherProcessesTimer: WATCHDOG_PID names the single
// process meant to feed the watchdog.
func TestWatchdogIgnoresAnotherProcessesTimer(t *testing.T) {
	t.Setenv("WATCHDOG_USEC", "120000000")
	t.Setenv("WATCHDOG_PID", strconv.Itoa(os.Getpid()+1))

	if got := WatchdogInterval(); got != 0 {
		t.Fatalf("interval = %s, want 0 when the watchdog belongs to another process", got)
	}

	t.Setenv("WATCHDOG_PID", strconv.Itoa(os.Getpid()))

	if got := WatchdogInterval(); got != 60*time.Second {
		t.Fatalf("interval = %s, want 60s for our own pid", got)
	}
}

// TestWatchdogKeepsFeeding is the behaviour the whole thing exists for.
func TestWatchdogKeepsFeeding(t *testing.T) {
	addr, messages := listen(t)

	t.Setenv("NOTIFY_SOCKET", addr)
	t.Setenv("WATCHDOG_USEC", "100000") // 100ms → fed every 50ms
	t.Setenv("WATCHDOG_PID", "")

	stop := make(chan struct{})
	defer close(stop)

	interval, running := StartWatchdog(stop)
	if !running {
		t.Fatal("the watchdog did not start")
	}

	if interval != 50*time.Microsecond*1000 {
		t.Fatalf("interval = %s, want 50ms", interval)
	}

	for i := 0; i < 3; i++ {
		select {
		case got := <-messages:
			if got != "WATCHDOG=1" {
				t.Fatalf("sent %q, want WATCHDOG=1", got)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("the watchdog stopped after %d pings", i)
		}
	}
}

// TestWatchdogStopsWhenAsked: a clean shutdown must not leave it running.
func TestWatchdogStopsWhenAsked(t *testing.T) {
	addr, messages := listen(t)

	t.Setenv("NOTIFY_SOCKET", addr)
	t.Setenv("WATCHDOG_USEC", "100000")
	t.Setenv("WATCHDOG_PID", "")

	stop := make(chan struct{})

	if _, running := StartWatchdog(stop); !running {
		t.Fatal("the watchdog did not start")
	}

	<-messages // one ping proves it is alive
	close(stop)

	// Drain anything already in flight, then expect silence.
	time.Sleep(300 * time.Millisecond)
	for len(messages) > 0 {
		<-messages
	}

	select {
	case got, ok := <-messages:
		if ok {
			t.Fatalf("the watchdog is still feeding after stop: %q", got)
		}
	case <-time.After(400 * time.Millisecond):
		// Silence is what we want.
	}
}

func TestStartWatchdogIsInertWithoutSystemd(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	t.Setenv("WATCHDOG_USEC", "")

	stop := make(chan struct{})
	defer close(stop)

	if _, running := StartWatchdog(stop); running {
		t.Fatal("the watchdog started with no service manager present")
	}
}
