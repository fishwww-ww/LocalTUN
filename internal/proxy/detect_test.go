package proxy

import (
	"errors"
	"net"
	"testing"
	"time"
)

type fakeConn struct{ net.Conn }

func (fakeConn) Close() error { return nil }

func TestDetectOverride(t *testing.T) {
	d := Detector{Dial: func(address string, timeout time.Duration) (net.Conn, error) {
		if address != "127.0.0.1:7897" {
			t.Fatalf("unexpected address %s", address)
		}
		return fakeConn{}, nil
	}}
	got, err := d.Detect("7897")
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:7897" {
		t.Fatalf("got %s", got)
	}
}

func TestDetectDefaultPorts(t *testing.T) {
	var seen []string
	d := Detector{Dial: func(address string, timeout time.Duration) (net.Conn, error) {
		seen = append(seen, address)
		if address == "127.0.0.1:1080" {
			return fakeConn{}, nil
		}
		return nil, errors.New("closed")
	}}
	got, err := d.Detect("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:1080" {
		t.Fatalf("got %s", got)
	}
	if len(seen) != 3 {
		t.Fatalf("seen %v", seen)
	}
}

func TestDetectNoProxy(t *testing.T) {
	d := Detector{Dial: func(address string, timeout time.Duration) (net.Conn, error) {
		return nil, errors.New("closed")
	}}
	if _, err := d.Detect(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeAddress(t *testing.T) {
	got, err := NormalizeAddress(":7897")
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:7897" {
		t.Fatalf("got %s", got)
	}
}
