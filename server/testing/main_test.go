package testing

import (
	"context"
	"github.com/google/uuid"
	pb "github.com/intiluha/grpcSandbox/grpcSandbox"
	"github.com/intiluha/grpcSandbox/server/internal"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"testing"
	"time"
)

const address = "localhost" + internal.Port

func TestCreateUser(t *testing.T) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewServiceExampleServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	user, err := client.CreateUser(ctx, createUserRequest)
	require.NoError(t, err)
	require.Equal(t, createUserRequest.Name, user.Name)
	require.Equal(t, createUserRequest.Age, user.Age)
	require.Equal(t, createUserRequest.UserType, user.UserType)
	require.Equal(t, user.UpdatedAt, user.CreatedAt)
	require.Equal(t, len(createUserRequest.Items), len(user.Items))
	for index, item := range user.Items {
		require.Equal(t, user.Id, item.UserId)
		require.Equal(t, createUserRequest.Items[index].Name, item.Name)
		require.Equal(t, item.UpdatedAt, item.CreatedAt)
	}
	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.Id})
	require.NoError(t, err)

	for _, request := range badCreateUserRequests {
		_, err = client.CreateUser(ctx, request)
		require.Error(t, err)
	}
}

func TestUpdateUser(t *testing.T) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewServiceExampleServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	user, err := client.CreateUser(ctx, createUserRequest)
	require.NoError(t, err)
	updatedUser, err := client.UpdateUser(ctx, &pb.UpdateUserRequest{
		Id:       user.Id,
		Name:     user.Name + "_updated",
		Age:      666,
		UserType: 0,
		Items: []*pb.UpdateItemRequest{{
			Id:   user.Items[0].Id,
			Name: user.Items[0].Name + "_updated",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, user.Id, updatedUser.Id)
	require.Equal(t, user.Name+"_updated", updatedUser.Name)
	require.Equal(t, int32(666), updatedUser.Age)
	require.Equal(t, user.UserType, updatedUser.UserType)
	require.Equal(t, user.CreatedAt, updatedUser.CreatedAt)
	require.NotEqual(t, user.UpdatedAt, updatedUser.UpdatedAt)
	require.Equal(t, len(user.Items), len(updatedUser.Items))
	require.Equal(t, user.Items[0].Id, updatedUser.Items[0].Id)
	require.Equal(t, user.Items[0].Name+"_updated", updatedUser.Items[0].Name)
	require.Equal(t, user.Items[0].UserId, updatedUser.Items[0].UserId)
	require.Equal(t, user.Items[0].CreatedAt, updatedUser.Items[0].CreatedAt)
	require.NotEqual(t, user.Items[0].UpdatedAt, updatedUser.Items[0].UpdatedAt)
	// foreign Item in UpdateUserRequest
	user1, err := client.CreateUser(ctx, createUserRequest)
	require.NoError(t, err)
	_, err = client.UpdateUser(ctx, &pb.UpdateUserRequest{
		Id: user.Id,
		Items: []*pb.UpdateItemRequest{{
			Id: user1.Items[0].Id,
		}},
	})
	require.Error(t, err)
	// fake User in UpdateUserRequest
	_, err = client.UpdateUser(ctx, &pb.UpdateUserRequest{Id: uuid.New().String()})
	require.Error(t, err)

	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.Id})
	require.NoError(t, err)
	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user1.Id})
	require.NoError(t, err)
}

func TestListUser(t *testing.T) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewServiceExampleServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	n := 5
	deleteUserRequests := make([]*pb.DeleteUserRequest, n)
	for i := 0; i < n; i++ {
		user, err := client.CreateUser(ctx, createUserRequest)
		require.NoError(t, err)
		deleteUserRequests[i] = &pb.DeleteUserRequest{Id: user.Id}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n-i+1; j++ {
			response, err := client.ListUser(ctx, &pb.ListUserRequest{PageFilter: &pb.PageFilter{
				Limit: uint32(i),
				Page:  uint32(j),
			}})
			require.NoError(t, err)
			require.Equal(t, i, len(response.Users))
		}
	}
	for _, request := range deleteUserRequests {
		_, err = client.DeleteUser(ctx, request)
		require.NoError(t, err)
	}
}

func TestGetUser(t *testing.T) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewServiceExampleServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	user0, err := client.CreateUser(ctx, createUserRequest)
	require.NoError(t, err)
	user1, err := client.GetUser(ctx, &pb.GetUserRequest{Id: user0.Id})
	require.NoError(t, err)
	require.Equal(t, user0, user1)

	_, err = client.GetUser(ctx, &pb.GetUserRequest{Id: uuid.New().String()})
	require.Error(t, err)

	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user0.Id})
	require.NoError(t, err)
}

func TestCreateItem(t *testing.T) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewServiceExampleServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	user, err := client.CreateUser(ctx, createUserRequest)
	require.NoError(t, err)
	for _, r := range createItemRequests {
		_, err := client.CreateItem(ctx, r)
		require.Error(t, err)

		r.UserId = user.Id
		item, err := client.CreateItem(ctx, r)
		require.NoError(t, err)
		require.Equal(t, r.Name, item.Name)
		require.Equal(t, user.Id, item.UserId)
		require.Equal(t, item.UpdatedAt, item.CreatedAt)
	}
	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.Id})
	require.NoError(t, err)

	_, err = client.CreateItem(ctx, badCreateItemRequest)
	require.Error(t, err)
}

func TestUpdateItem(t *testing.T) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer conn.Close()
	client := pb.NewServiceExampleServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	user, err := client.CreateUser(ctx, createUserRequest)
	require.NoError(t, err)
	for _, item := range user.Items {
		updatedItem, err := client.UpdateItem(ctx, &pb.UpdateItemRequest{
			Id:   item.Id,
			Name: item.Name + "_updated",
		})
		require.NoError(t, err)
		require.Equal(t, item.Id, updatedItem.Id)
		require.Equal(t, item.Name+"_updated", updatedItem.Name)
		require.Equal(t, item.UserId, updatedItem.UserId)
		require.Equal(t, item.CreatedAt, updatedItem.CreatedAt)
		require.NotEqual(t, item.UpdatedAt, updatedItem.UpdatedAt)
	}

	_, err = client.UpdateItem(ctx, &pb.UpdateItemRequest{Id: uuid.New().String()})
	require.Error(t, err)

	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.Id})
	require.NoError(t, err)
}

// Any test above can be thought of as TestDeleteUser
