package app

import (
	"net"
	"strings"
	"testing"
	"testing/fstest"
)

func TestBusyPortFallback(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port

	listener, err := listenLocal(port, true)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().(*net.TCPAddr)
	if !address.IP.IsLoopback() || address.Port == port {
		t.Fatalf("expected another loopback port, got %v", address)
	}

	listener, err = listenLocal(port, false)
	if listener != nil {
		listener.Close()
		t.Fatal("explicit busy port must fail")
	}
	if err == nil || !strings.Contains(err.Error(), "--port 0") {
		t.Fatalf("expected actionable port error, got %v", err)
	}
}

func TestInvalidPortDoesNotFallBack(t *testing.T) {
	listener, err := listenLocal(-1, true)
	if listener != nil {
		listener.Close()
		t.Fatal("invalid port unexpectedly started a server")
	}
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestStartupExplainsMissingAndInvalidSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name string
		data fstest.MapFS
		raw  bool
		want string
	}{
		{"first setup", fstest.MapFS{}, true, "UC Print setup is not complete"},
		{"broken local data", fstest.MapFS{"manifest.json": {Data: []byte("broken")}}, true, "local snapshot failed validation"},
		{"broken release", fstest.MapFS{}, false, "embedded snapshot is damaged"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// An invalid port ensures this test cannot accidentally start a listener.
			err := Run(tc.data, fstest.MapFS{}, tc.raw, []string{"--port", "-1"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wrong startup result: %v", err)
			}
			if tc.name == "first setup" && (!strings.Contains(err.Error(), "does not create one") || !strings.Contains(err.Error(), "ucsync import")) {
				t.Fatal("missing recovery instructions")
			}
		})
	}
}
