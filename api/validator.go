package api

import (
	"SimpleBankProject/db/util"

	"github.com/go-playground/validator/v10"
)

// validCurrency will take a fieldLevel and return true when it is validated
// validator.FieldLevel - FieldLevel is an interface containing all the information and helper functions to validate a field
var validCurrency validator.Func = func(fieldLevel validator.FieldLevel) bool {
	// obtain value of field - this is a reflection value so we have to call Interface()
	// which will give us the value as an empty interface
	// finally we convert the value to a string
	if currency, ok := fieldLevel.Field().Interface().(string); ok {
		// if ok is true, check if currency is supported
		return util.IsSupportedCurrency(currency)
	}
	// else field is not a string
	return false
}
