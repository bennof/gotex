package session

import (
	"os"
	"path/filepath"
	"time"

	"github.com/bennof/gotex/simpleserver"
)

type SessionFS struct {
	Session
	path    string
	maxSize int64
}

func NewSessionFS(rootDir string, maxSize int64) func(id string, now time.Time) (*SessionFS, error) {
	return func(id string, now time.Time) (*SessionFS, error) {
		s := &SessionFS{
			Session: Session{
				id:        id,
				createdAt: now,
				lastSeen:  now,
			},
			path:    filepath.Join(rootDir, id),
			maxSize: maxSize,
		}
		if err := os.MkdirAll(s.path, os.ModePerm); err != nil {
			return nil, err
		}
		return s, nil
	}
}

func (s *SessionFS) Close() error {
	if err := os.RemoveAll(s.path); err != nil {
		return err
	}
	return nil
}

func (s *SessionFS) Path() string {
	return s.path
}

func (s *SessionFS) PathJoin(path string) string {
	return filepath.Join(s.path, path)
}

func (s *SessionFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(s.path, path))
}

func (s *SessionFS) DirSize() (int64, error) {
	return simpleserver.DirSize(s.path)
}
