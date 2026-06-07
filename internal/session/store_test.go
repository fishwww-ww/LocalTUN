package session

import (
	"testing"
	"time"
)

func TestStoreSaveLoadListRemove(t *testing.T) {
	store := NewStore(t.TempDir())
	meta := Metadata{
		ID:         "test-1",
		Target:     "root@gpu01",
		User:       "root",
		Host:       "gpu01",
		SSHPort:    22,
		LocalProxy: "127.0.0.1:7897",
		RemotePort: 46327,
		PID:        123,
		CreatedAt:  time.Now().UTC(),
		ProxyURL:   "http://127.0.0.1:46327",
	}
	if err := store.Save(meta); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != meta.ID || got.RemotePort != meta.RemotePort {
		t.Fatalf("Load = %#v", got)
	}
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != meta.ID {
		t.Fatalf("List = %#v", list)
	}
	if err := store.Remove(meta.ID); err != nil {
		t.Fatal(err)
	}
	list, err = store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("List after remove = %#v", list)
	}
}

func TestStoreRejectsBadID(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Save(Metadata{ID: "../bad"}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := store.Load("../bad"); err == nil {
		t.Fatal("expected error")
	}
}
