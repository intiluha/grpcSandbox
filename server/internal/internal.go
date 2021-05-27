package internal

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const Port = ":50052"

var (
	ErrEmptyUserName   = status.Error(codes.InvalidArgument, "user name can't be empty")
	ErrEmptyAge        = status.Error(codes.InvalidArgument, "user age can't be empty or zero")
	ErrInvalidUserType = status.Error(codes.InvalidArgument, "invalid user type")
	ErrEmptyItemName   = status.Error(codes.InvalidArgument, "item name can't be empty")
	ErrSomeItemNotFound = status.Error(codes.NotFound, "item not found")
)

func ErrUserNotFound(id string) error {
	return status.Errorf(codes.NotFound, "user with id {%s} not found", id)
}

func ErrItemNotFound(id string) error {
	return status.Errorf(codes.NotFound, "item with id {%s} not found", id)
}
