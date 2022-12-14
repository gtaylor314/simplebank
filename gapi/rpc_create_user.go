package gapi

import (
	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/db/util"
	"SimpleBankProject/pb"
	"SimpleBankProject/val"
	"context"

	"github.com/lib/pq"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	// validate that the CreateUserRequest properties meet the criteria defined in validator.go
	violations := validateCreateUserRequest(req)
	// if violations isn't nil, then at least one property has failed to meet the criteria
	if violations != nil {
		// invalidArgumentError defined in error.go
		return nil, invalidArgumentError(violations)
	}

	// GetPassword offers a check for nil values which is better than simply grabbing the password from req.Password.
	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
	}

	// if no err, begin user creation
	arg := db.CreateUserParams{
		Username:       req.GetUsername(),
		HashedPassword: hashedPassword,
		FullName:       req.GetFullName(),
		Email:          req.GetEmail(),
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		// try to convert err to type pq.Error
		// this is to provide a better error in the event that someone attempts to create a user with a
		// username or email that already exists
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return nil, status.Errorf(codes.AlreadyExists, "user already exists: %s", &pqErr)
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}

	rsp := &pb.CreateUserResponse{
		User: convertUser(user),
	}

	return rsp, nil
}

// validateCreateUserRequest will validate each property of the CreateUserRequest object - it will return a slice of errors
// specifically a slice of BadRequest_FieldViolation struct from the errdetails package of gRPC (named returned violations)
func validateCreateUserRequest(req *pb.CreateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateUsername(req.GetUsername()); err != nil {
		violations = append(violations, fieldViolation("username", err))
	}
	if err := val.ValidatePassword(req.GetPassword()); err != nil {
		violations = append(violations, fieldViolation("password", err))
	}
	if err := val.ValidateFullName(req.GetFullName()); err != nil {
		violations = append(violations, fieldViolation("full_name", err))
	}
	if err := val.ValidateEmail(req.GetEmail()); err != nil {
		violations = append(violations, fieldViolation("email", err))
	}
	return violations
}
