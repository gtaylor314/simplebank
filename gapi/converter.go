package gapi

import (
	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// converting from a db.User object to a pb.User object - this separates the db layer from the api layer
func convertUser(user db.User) *pb.User {
	return &pb.User{
		Username:         user.Username,
		FullName:         user.FullName,
		Email:            user.Email,
		PasswordChangeAt: timestamppb.New(user.PasswordChangeAt),
		CreatedAt:        timestamppb.New(user.CreatedAt),
	}
}
