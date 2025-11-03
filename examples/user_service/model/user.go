package model

import (
	"errors"
	"time"
)

type User struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Birth     time.Time `json:"birth,omitempty"`
	Active    bool      `json:"active"`
	Uuid      string    `json:"uuid"`
}

func (u *User) Validate() error {
	if u.FirstName == "" {
		return errors.New("FirstName is required")
	}
	if u.LastName == "" {
		return errors.New("LastName is required")
	}
	if u.Email == "" {
		// TODO : validate the email format
		return errors.New("Email is required")
	}
	return nil
}
