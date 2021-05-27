package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	pb "github.com/intiluha/grpcSandbox/grpcSandbox"
	"github.com/intiluha/grpcSandbox/server/internal"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"net"
	"os"
	"time"
)

var (
	role     = os.Getenv("GRPC_SANDBOX_ROLE")
	password = os.Getenv("GRPC_SANDBOX_PASSWORD")
	dbName   = os.Getenv("GRPC_SANDBOX_DB_NAME")
	db, err  = sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", role, password, dbName))
)

func getItem(ctx context.Context, id string) (*pb.Item, error) {
	item := pb.Item{
		Id: id,
	}
	query := `select name, user_id, created_at, updated_at from items where id=$1`
	var createdAt, updatedAt time.Time
	err := db.QueryRowContext(ctx, query, id).Scan(&item.Name, &item.UserId, &createdAt, &updatedAt)
	item.CreatedAt, item.UpdatedAt = timestamppb.New(createdAt), timestamppb.New(updatedAt)
	if err == sql.ErrNoRows {
		return nil, internal.ErrItemNotFound(id)
	}
	return &item, err
}

func updateItemStructure(i *pb.Item, r *pb.UpdateItemRequest, updatedAt *timestamppb.Timestamp) {
	i.UpdatedAt = updatedAt
	if r.Name != "" {
		i.Name = r.Name
	}
}

func updateItemInTable(ctx context.Context, i *pb.Item, updatedAt time.Time) error {
	query := `update items set name=$1, updated_at=$2 where id=$3;`
	_, err := db.ExecContext(ctx, query, i.Name, updatedAt, i.Id)
	return err
}

func createItemInTable(ctx context.Context, i *pb.Item, createdAt time.Time) error {
	query := `insert into items values ($1, $2, $3, $4, $5);`
	_, err := db.ExecContext(ctx, query, i.Id, i.Name, i.UserId, createdAt, createdAt)
	return err
}

func updateUserStructure(u *pb.User, r *pb.UpdateUserRequest, updatedAt *timestamppb.Timestamp) []int {
	u.UpdatedAt = updatedAt
	if r.Name != "" {
		u.Name = r.Name
	}
	if r.Age != 0 {
		u.Age = r.Age
	}
	if r.UserType != pb.UserType_INVALID_USER_TYPE {
		u.UserType = r.UserType
	}
	updates := make(map[string]*pb.UpdateItemRequest)
	for _, item := range r.Items {
		updates[item.Id] = item
	}
	indices := make([]int, 0, len(r.Items))
	for index, item := range u.Items {
		if update, exist := updates[item.Id]; exist {
			indices = append(indices, index)
			updateItemStructure(item, update, updatedAt)
		}
	}
	return indices
}

func updateUserInTable(ctx context.Context, u *pb.User, updatedAt time.Time) error {
	query := `update users set name=$1, age=$2, type=$3, updated_at=$4 where id=$5;`
	_, err := db.ExecContext(ctx, query, u.Name, u.Age, u.UserType, updatedAt, u.Id)
	return err
}

func createUserInTable(ctx context.Context, u *pb.User, createdAt time.Time) error {
	query := `insert into users values ($1, $2, $3, $4, $5, $6);`
	_, err := db.ExecContext(ctx, query, u.Id, u.Name, u.Age, u.UserType, createdAt, createdAt)
	return err
}

func populateItems(ctx context.Context, user *pb.User) error {
	var createdAt, updatedAt time.Time
	selectItemsQuery := `select id, name, created_at, updated_at from items where user_id=$1`
	itemRows, err := db.QueryContext(ctx, selectItemsQuery, user.Id)
	if err != nil {
		return err
	}
	for itemRows.Next() {
		item := &pb.Item{UserId: user.Id}
		if err = itemRows.Scan(&item.Id, &item.Name, &createdAt, &updatedAt); err != nil {
			return err
		}
		item.CreatedAt, item.UpdatedAt = timestamppb.New(createdAt), timestamppb.New(updatedAt)
		user.Items = append(user.Items, item)
	}
	return itemRows.Err()
}

// server is used to implement pb.ServiceExampleServiceServer
type server struct {
	pb.UnimplementedServiceExampleServiceServer
}

func (s server) CreateUser(ctx context.Context, r *pb.CreateUserRequest) (*pb.User, error) {
	createdAt := timestamppb.Now()
	// We lower timestamp precision to that supported by the database
	createdAt.Nanos -= createdAt.Nanos % 1000
	if r.Name == "" {
		return nil, internal.ErrEmptyUserName
	}
	if r.Age == 0 {
		return nil, internal.ErrEmptyAge
	}
	if r.UserType == pb.UserType_INVALID_USER_TYPE {
		return nil, internal.ErrInvalidUserType
	}
	user := &pb.User{
		Id:        uuid.New().String(),
		Name:      r.Name,
		Age:       r.Age,
		UserType:  r.UserType,
		Items:     make([]*pb.Item, len(r.Items)),
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	if err := createUserInTable(ctx, user, createdAt.AsTime()); err != nil {
		return nil, err
	}
	for index, itemRequest := range r.Items {
		// At the time of calling CreateUser, client can't know user_id, so we have to correct it
		itemRequest.UserId = user.Id
		user.Items[index], err = s.CreateItem(ctx, itemRequest)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (s server) UpdateUser(ctx context.Context, r *pb.UpdateUserRequest) (*pb.User, error) {
	updatedAt := timestamppb.Now()
	// We lower timestamp precision to that supported by the database
	updatedAt.Nanos -= updatedAt.Nanos % 1000
	user, err := s.GetUser(ctx, &pb.GetUserRequest{Id: r.Id})
	if err != nil {
		return nil, err
	}
	indices := updateUserStructure(user, r, updatedAt)
	// If len(indices) < len(r.Items), then some items from r not present in user
	if len(indices) < len(r.Items) {
		return nil, internal.ErrSomeItemNotFound
	}

	// Update database
	if err = updateUserInTable(ctx, user, updatedAt.AsTime()); err != nil {
		return nil, err
	}
	for _, index := range indices {
		if err = updateItemInTable(ctx, user.Items[index], updatedAt.AsTime()); err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (s server) DeleteUser(ctx context.Context, r *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	query := `delete from users where id=$1;`
	_, err := db.ExecContext(ctx, query, r.Id)
	return &pb.DeleteUserResponse{}, err
}

func (s server) ListUser(ctx context.Context, r *pb.ListUserRequest) (*pb.ListUserResponse, error) {
	response := &pb.ListUserResponse{
		Users: make([]*pb.User, 0, r.PageFilter.Limit),
	}
	selectUsersQuery := `select * from users order by id limit $1 offset $2`
	userRows, err := db.QueryContext(ctx, selectUsersQuery, r.PageFilter.Limit, r.PageFilter.Page)
	if err != nil {
		return nil, err
	}

	var createdAt, updatedAt time.Time
	for userRows.Next() {
		user := &pb.User{
			Items: make([]*pb.Item, 0),
		}
		if err = userRows.Scan(&user.Id, &user.Name, &user.Age, &user.UserType, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		user.CreatedAt, user.UpdatedAt = timestamppb.New(createdAt), timestamppb.New(updatedAt)

		if err = populateItems(ctx, user); err != nil {
			return nil, err
		}
		response.Users = append(response.Users, user)
	}
	return response, userRows.Err()
}

func (s server) GetUser(ctx context.Context, r *pb.GetUserRequest) (*pb.User, error) {
	user := &pb.User{
		Id:    r.Id,
		Items: make([]*pb.Item, 0),
	}
	selectUserQuery := `select name, age, type, created_at, updated_at from users where id=$1`
	var createdAt, updatedAt time.Time
	err := db.QueryRowContext(ctx, selectUserQuery, user.Id).Scan(&user.Name, &user.Age, &user.UserType, &createdAt, &updatedAt)
	user.CreatedAt, user.UpdatedAt = timestamppb.New(createdAt), timestamppb.New(updatedAt)
	if err == sql.ErrNoRows {
		return nil, internal.ErrUserNotFound(user.Id)
	}
	if err != nil {
		return nil, err
	}
	return user, populateItems(ctx, user)
}

func (s server) CreateItem(ctx context.Context, r *pb.CreateItemRequest) (*pb.Item, error) {
	createdAt := timestamppb.Now()
	// We lower timestamp precision to that supported by the database
	createdAt.Nanos -= createdAt.Nanos % 1000
	if r.Name == "" {
		return nil, internal.ErrEmptyItemName
	}
	item := &pb.Item{
		Id:        uuid.New().String(),
		Name:      r.Name,
		UserId:    r.UserId,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	return item, createItemInTable(ctx, item, createdAt.AsTime())
}

func (s server) UpdateItem(ctx context.Context, r *pb.UpdateItemRequest) (*pb.Item, error) {
	updatedAt := timestamppb.Now()
	// We lower timestamp precision to that supported by the database
	updatedAt.Nanos -= updatedAt.Nanos % 1000
	item, err := getItem(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	updateItemStructure(item, r, updatedAt)
	return item, updateItemInTable(ctx, item, updatedAt.AsTime())
}

func main() {
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("Connected!")

	lis, err := net.Listen("tcp", internal.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterServiceExampleServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// TODO: Doc strings

//	(1)	Maybe time of update and creation should be computed locally instead of once per request.
//		This will simplify code, but this way items created with CreateUser will get slightly
//		different time then user itself (and other items)
//	(2)	Maybe null name, age and/or user_type should be allowed
//	(3)	Maybe no update should be performed (no change to updated_at) if no field need an update
