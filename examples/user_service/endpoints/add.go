package endpoints

import (
	"fmt"

	"github.com/hypersequent/uuid7"
	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/user_service/model"
	"github.com/telemac/natsservice/examples/user_service/pkg/user_store"
)

var _ natsservice.Endpointer = (*UserAddEndpoint)(nil)

type UserAddRequest struct {
	User model.User `json:"user"`
}

type UserAddResponse struct {
	UUID string `json:"uuid"`
}

type UserAddEndpoint struct {
	natsservice.Endpoint
	userStore user_store.UserStore
}

func NewUserAddEndpoint(userStore user_store.UserStore) *UserAddEndpoint {
	return &UserAddEndpoint{
		userStore: userStore,
	}
}

func (e *UserAddEndpoint) Config() *natsservice.EndpointConfig {
	serviceName := e.Service().Config().Name
	return &natsservice.EndpointConfig{
		Name: "add",
		Metadata: map[string]string{
			"description": "adds a new user",
			"service":     e.Service().Config().Name,
			"version":     "1.0.0",
			"author":      "telemac",
		},
		QueueGroup: serviceName + ".add",
	}
}

func (e *UserAddEndpoint) Handle(request micro.Request) {
	defer natsservice.RecoverPanic(e, request)

	log := e.Service().Logger().With(
		"endpoint", e.Config().Name,
		"version", e.Service().Config().Version,
	)

	userAddRequest, err := natsservice.UnmarshalRequestWithLog[UserAddRequest](request, log)
	if err != nil {
		return
	}
	err = userAddRequest.User.Validate()
	if err != nil {
		log.Warn("user is invalid", "error", err)
		description := fmt.Sprintf("user is invalid: %s", err)
		request.Error("500", description, nil)
		return
	}

	uuid := uuid7.NewString()
	userAddRequest.User.Uuid = uuid

	err = e.userStore.Add(&userAddRequest.User)
	if err != nil {
		request.Error("500", "add user failed", nil)
		log.Error("adding user failed", "error", err)
		return
	}

	response := UserAddResponse{
		UUID: uuid,
	}
	log.Info("adding user", "user", userAddRequest.User)
	request.RespondJSON(response)
}
