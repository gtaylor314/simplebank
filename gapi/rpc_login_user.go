package gapi

import (
	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/db/util"
	"SimpleBankProject/pb"
	"SimpleBankProject/val"
	"context"
	"database/sql"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	// validate the LoginUserRequest properties meet the criteria outlined in validator.go
	violations := validateLoginUserRequest(req)
	// if violations isn't nil, at least one property has failed to meet the criteria
	if violations != nil {
		// invalidArgumentError defined in error.go
		return nil, invalidArgumentError(violations)
	}

	// get user requested if it exists - GetUsername checks for nil which is better than just using req.Username
	user, err := server.store.GetUser(ctx, req.GetUsername())
	if err != nil {
		// two reasons err may not be nil
		// first, the user doesn't exist
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "failed to find user: %s", err)
		}
		// second, internal issue with the GetUser api call
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	// check if the password provided is correct - GetPassword() checks for nil
	err = util.CheckPassword(req.GetPassword(), user.HashedPassword)
	if err != nil {
		// if err isn't nil, the password provided was incorrect
		return nil, status.Errorf(codes.PermissionDenied, "password provided is incorrect: %s", err)
	}

	// user exists and password provided is correct, create access token
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(user.Username, server.config.AccessTokenDuration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create access token: %s", err)
	}

	// create refresh token with a longer valid duration than the access token - will use to create session
	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user.Username, server.config.RefreshTokenDuration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create refresh token: %s", err)
	}

	// pass context for metadata extraction - allows us to populate UserAgent and ClientIP in the session
	mtdt := server.extractMetadata(ctx)

	// create session
	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    mtdt.UserAgent, // client type
		ClientIp:     mtdt.ClientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %s", err)
	}

	rsp := &pb.LoginUserResponse{
		Login: &pb.Login{
			SessionId:             session.ID.String(),
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  timestamppb.New(accessPayload.ExpiredAt),
			RefreshToken:          refreshToken,
			RefreshTokenExpiresAt: timestamppb.New(refreshPayload.ExpiredAt),
			User:                  convertUser(user),
		},
	}
	return rsp, nil
}

// validateLoginUserRequest will validate each property of the LoginUserRequest object - it will return a slice of errors
// specifically a slice of BadRequest_FieldViolation struct from the errdetails package of gRPC (named returned violations)
func validateLoginUserRequest(req *pb.LoginUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateUsername(req.GetUsername()); err != nil {
		violations = append(violations, fieldViolation("username", err))
	}
	if err := val.ValidatePassword(req.GetPassword()); err != nil {
		violations = append(violations, fieldViolation("password", err))
	}

	return violations
}
