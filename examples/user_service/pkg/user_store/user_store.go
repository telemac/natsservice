package user_store

import "github.com/telemac/natsservice/examples/user_service/model"

type UserStore interface {
	Add(user *model.User) error
	Get(email string) (model.User, error)
}
