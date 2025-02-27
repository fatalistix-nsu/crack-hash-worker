package validation

import (
	"fmt"
	"github.com/go-playground/validator/v10"
)

type RequestValidator struct {
	v *validator.Validate
}

func (rv *RequestValidator) Validate(i interface{}) error {
	return rv.v.Struct(i)
}

func NewRequestValidator() (*RequestValidator, error) {
	const op = "validation.NewCustomValidator"

	v := validator.New()

	if err := v.RegisterValidation("uniquechars", uniqueChars); err != nil {
		return nil, fmt.Errorf(`%s: error registering "uniquechars" validator: %w`, op, err)
	}

	if err := v.RegisterValidation("md5hash", md5Hash); err != nil {
		return nil, fmt.Errorf(`%s: error registering "md5hash" validator: %w`, op, err)
	}

	return &RequestValidator{
		v: v,
	}, nil
}
