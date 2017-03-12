package mem

import (
	"fmt"
	"sync"
)

import (
)

import (
	"github.com/timtadh/dynagrok/localize/discflo/web/models"
)


type SessionMapStore struct {
	lock sync.Mutex
	name string
	store map[string]*models.Session
}

func NewSessionMapStore(name string) *SessionMapStore {
	return &SessionMapStore{
		name: name,
		store: make(map[string]*models.Session, 1000),
	}
}

func (m *SessionMapStore) Name() string {
	return m.name
}

func (m *SessionMapStore) Get(key string) (*models.Session, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if s, has := m.store[key]; has {
		return s.Copy(), nil
	}
	return nil, fmt.Errorf("Session not in store")
}

func (m *SessionMapStore) Invalidate(s *models.Session) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.store, s.Key)
	return nil
}

func (m *SessionMapStore) Update(s *models.Session) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if s == nil {
		return fmt.Errorf("passed in a nil session")
	}
	m.store[s.Key] = s.Copy()
	return nil
}

