package utils

import (
	"time"

	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	Validator *validator.Validate
}

func NewCustomValidator() *CustomValidator {
	v := validator.New()
	v.RegisterValidation("after", func(fl validator.FieldLevel) bool {
		startTime, ok := fl.Field().Interface().(time.Time)
		return ok && startTime.After(time.Now())
	})
	return &CustomValidator{
		Validator: v,
	}
}
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.Validator.Struct(i)
}
