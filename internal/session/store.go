package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

type Metadata struct {
	ID         string    `json:"id"`
	Target     string    `json:"target"`
	User       string    `json:"user"`
	Host       string    `json:"host"`
	SSHPort    int       `json:"ssh_port"`
	Identity   string    `json:"identity"`
	LocalProxy string    `json:"local_proxy"`
	RemotePort int       `json:"remote_port"`
	PID        int       `json:"pid"`
	CreatedAt  time.Time `json:"created_at"`
	ProxyURL   string    `json:"proxy_url"`
}

type Store struct {
	dir string
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func DefaultStore() (Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Store{}, err
	}
	return NewStore(filepath.Join(home, ".localtun-next", "sessions")), nil
}

func NewStore(dir string) Store {
	return Store{dir: dir}
}

func (s Store) Dir() string {
	return s.dir
}

func (s Store) Save(meta Metadata) error {
	if err := validateID(meta.ID); err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("创建 session 目录失败: %w", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(meta.ID), data, 0644)
}

func (s Store) Load(id string) (Metadata, error) {
	if err := validateID(id); err != nil {
		return Metadata{}, err
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return Metadata{}, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

func (s Store) List() ([]Metadata, error) {
	files, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Metadata
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		id := file.Name()[:len(file.Name())-len(".json")]
		meta, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s Store) Remove(id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	err := os.Remove(s.path(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func validateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("非法 session id: %s", id)
	}
	return nil
}
