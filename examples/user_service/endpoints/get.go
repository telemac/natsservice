package endpoints

import (
	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/natsservice"
	"github.com/telemac/natsservice/examples/user_service/model"
	"github.com/telemac/natsservice/examples/user_service/pkg/user_store"
)

var _ natsservice.Endpointer = (*UserGetEndpoint)(nil)

type UserGetRequest struct {
	UUID string `json:"uuid"`
}

type UserGetResponse struct {
	User model.User `json:"user"`
}

type UserGetEndpoint struct {
	natsservice.Endpoint
	userStore user_store.UserStore
}

func NewUserGetEndpoint(userStore user_store.UserStore) *UserGetEndpoint {
	return &UserGetEndpoint{
		userStore: userStore,
	}
}

func (e *UserGetEndpoint) Config() *natsservice.EndpointConfig {
	serviceName := e.Service().Config().Name
	return &natsservice.EndpointConfig{
		Name: "get",
		Metadata: map[string]string{
			"description": "gets a user by uuid",
			"service":     e.Service().Config().Name,
			"version":     "1.0.0",
			"author":      "telemac",
		},
		QueueGroup: serviceName + ".get",
	}
}

func (e *UserGetEndpoint) Handle(request micro.Request) {
	defer natsservice.RecoverPanic(e, request)

	log := e.Service().Logger().With(
		"endpoint", e.Config().Name,
		"version", e.Service().Config().Version,
	)

	userGetRequest, err := natsservice.UnmarshalRequestWithLog[UserGetRequest](request, log)
	if err != nil {
		return
	}

	if userGetRequest.UUID == "" {
		log.Warn("uuid is required")
		request.Error("400", "uuid is required", nil)
		return
	}

	user, err := e.userStore.Get(userGetRequest.UUID)
	if err != nil {
		log.Error("getting user failed", "error", err, "uuid", userGetRequest.UUID)
		request.Error("500", "getting user failed", nil)
		return
	}

	response := UserGetResponse{
		User: user,
	}
	log.Info("getting user", "uuid", userGetRequest.UUID)
	request.RespondJSON(response)
}
