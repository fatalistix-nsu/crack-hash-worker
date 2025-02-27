package validation

import "github.com/go-playground/validator/v10"

func uniqueChars(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	visited := make(map[rune]bool)

	for _, r := range str {
		if _, ok := visited[r]; ok {
			return false
		}

		visited[r] = true
	}

	return true
}
