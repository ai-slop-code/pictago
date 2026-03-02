package utils

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

func notBlank(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	return strings.TrimSpace(str) != ""
}

func NewValidator() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	_ = validate.RegisterValidation("notblank", notBlank)
	return validate
}
