package gapi

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fieldViolation returns the BadRequest_FieldViolation struct with the field name (username, password, full name, etc.) and
// the error object (use err.Error() to turn the error in err into a string for the description property)
func fieldViolation(field string, err error) *errdetails.BadRequest_FieldViolation {
	return &errdetails.BadRequest_FieldViolation{
		Field:       field,
		Description: err.Error(),
	}
}

// invalidArgumentError takes any violations and provides the error code and status message with details as an error
func invalidArgumentError(violations []*errdetails.BadRequest_FieldViolation) error {
	// badRequest holds the field violations data
	badRequest := &errdetails.BadRequest{FieldViolations: violations}
	// statusInvalid holds the Invalid Argument code and the status message
	statusInvalid := status.New(codes.InvalidArgument, "invalid parameters")
	// WithDetails returns a new status object with additional details from the badRequest object
	statusDetails, err := statusInvalid.WithDetails(badRequest)
	if err != nil {
		// if err isn't nil, then something went wrong with grabbing details from the badRequest object
		// return a nil CreateUserResponse and the statusInvalid error
		return statusInvalid.Err()
	}
	// return a nil CreateUserResponse and the statusDetails error
	return statusDetails.Err()
}
