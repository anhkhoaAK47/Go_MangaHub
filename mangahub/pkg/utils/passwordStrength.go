package utils

import "unicode"

func IsPasswordStrong(password string) bool{
	var (
		hasMinLen = false
		hasUpper = false
		hasLower = false
		hasNumber = false
	)


	if len(password) >= 8 {
		hasMinLen = true
	}

	for _, char := range password {
		switch {
			case unicode.IsUpper(char):
				hasUpper = true
			case unicode.IsLower(char):
				hasLower = true
			case unicode.IsNumber(char):
				hasNumber = true
		}
	}

	return hasMinLen && hasLower && hasUpper && hasNumber
}