package testing

import (
	pb "github.com/intiluha/grpcSandbox/grpcSandbox"
)

var (
	createItemRequests = []*pb.CreateItemRequest{
		{
			Name:   "cat",
		},
		{
			Name:   "dog",
		},
	}
	badCreateItemRequest = &pb.CreateItemRequest{}
	createUserRequest = &pb.CreateUserRequest{
		Name:     "John Doe",
		Age:      14,
		UserType: pb.UserType_CUSTOMER_USER_TYPE,
		Items:    createItemRequests,
		Test:     123,
	}
	badCreateUserRequests = []*pb.CreateUserRequest{
		{
			Age:      14,
			UserType: pb.UserType_CUSTOMER_USER_TYPE,
			Items:    createItemRequests,
			Test:     123,
		},
		{
			Name:     "John Doe",
			UserType: pb.UserType_CUSTOMER_USER_TYPE,
			Items:    createItemRequests,
			Test:     123,
		},
		{
			Name:     "John Doe",
			Age:      14,
			Items:    createItemRequests,
			Test:     123,
		},
	}
)
