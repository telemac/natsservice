package user_store

import (
	"errors"
	"fmt"
	"sync"

	"github.com/telemac/natsservice/examples/user_service/model"
)

var _ UserStore = (*MemoryUserStore)(nil)

type MemoryUserStore struct {
	users map[string]model.User
	mutex sync.RWMutex
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users: make(map[string]model.User),
	}
}

func (store *MemoryUserStore) Add(user *model.User) error {
	err := user.Validate()
	if err != nil {
		return fmt.Errorf("User validation failed: %v", err)
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.users[user.Email] = *user
	return nil
}

func (store *MemoryUserStore) Get(email string) (model.User, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	user, ok := store.users[email]
	if !ok {
		return model.User{}, errors.New("User not found")
	}
	return user, nil
}
