package next

import "testing"

type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }

func TestListenerPort(t *testing.T) {
	port, err := listenerPort(fakeAddr("127.0.0.1:46327"))
	if err != nil {
		t.Fatal(err)
	}
	if port != 46327 {
		t.Fatalf("port = %d", port)
	}
}
