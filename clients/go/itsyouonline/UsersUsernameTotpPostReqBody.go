package itsyouonline

import (
	"gopkg.in/validator.v2"
)

type UsersUsernameTotpPostReqBody struct {
	Totpcode string `json:"totpcode" validate:"nonzero"`
}

func (s UsersUsernameTotpPostReqBody) Validate() error {

	return validator.Validate(s)
}
