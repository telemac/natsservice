package user_store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/telemac/natsservice/examples/user_service/model"
	"github.com/telemac/natsservice/pkg/keyvalue"
)

var _ UserStore = (*KvUserStore)(nil)

type KvUserStore struct {
	kv  keyvalue.KeyValuer
	ctx context.Context
}

func NewKvUserStore(ctx context.Context, kv keyvalue.KeyValuer) *KvUserStore {
	return &KvUserStore{
		ctx: ctx,
		kv:  kv,
	}
}

func (store *KvUserStore) Add(user *model.User) error {
	err := user.Validate()
	if err != nil {
		return err
	}

	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	// Store user by UUID
	return store.kv.Set(store.ctx, "user."+user.Uuid, userData)
}

func (store *KvUserStore) Get(uuid string) (model.User, error) {
	// Then get user by UUID
	userData, err := store.kv.Get(store.ctx, "user."+uuid)
	if err != nil {
		if errors.Is(err, keyvalue.ErrKeyNotFound) {
			return model.User{}, errors.New("user not found")
		}
		return model.User{}, err
	}

	var user model.User
	err = json.Unmarshal(userData, &user)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}
