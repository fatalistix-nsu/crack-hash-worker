package validation

import (
	"github.com/go-playground/validator/v10"
	"regexp"
)

var md5Regex = regexp.MustCompile("^[a-fA-F0-9]{32}$")

func md5Hash(fl validator.FieldLevel) bool {
	return md5Regex.Match([]byte(fl.Field().String()))
}
