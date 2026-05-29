
package server

import (
	"context"
	pb "github.com/yourname/proto/user"
	"user/internal/logic"
	"user/internal/svc"
	"user/internal/types"
)

type UserGrpcServer struct {
	pb.UnimplementedUserServiceServer
	logic *logic.Logic
}

func NewUserGrpcServer(svcCtx *svc.ServiceContext) *UserGrpcServer {
	return &UserGrpcServer{
		logic: logic.NewLogic(svcCtx),
	}
}

// Register 用户注册
func (s *UserGrpcServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	err := s.logic.Register(&types.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Phone:    req.Phone,
	})
	if err != nil {
		return &pb.RegisterResponse{
			Success: false,
			Message: err.Error(),
		}, err
	}

	return &pb.RegisterResponse{
		Success: true,
		Message: "register success",
	}, nil
}

// Login 用户登录
func (s *UserGrpcServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	resp, err := s.logic.Login(&types.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return &pb.LoginResponse{
			Message: err.Error(),
		}, err
	}

	return &pb.LoginResponse{
		Token:   resp.Token,
		Message: "login success",
	}, nil
}

// GetUserList 获取用户列表
func (s *UserGrpcServer) GetUserList(ctx context.Context, req *pb.GetUserListRequest) (*pb.GetUserListResponse, error) {
	resp, err := s.logic.GetUserList(&types.PageRequest{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	// 转换用户列表
	users := make([]*pb.User, 0)
	for _, u := range resp.List.([]types.User) {
		users = append(users, &pb.User{
			Id:        uint64(u.ID),
			Username:  u.Username,
			Email:     u.Email,
			Phone:     u.Phone,
			Status:    int32(u.Status),
			CreatedAt: u.CreatedAt.Unix(),
			UpdatedAt: u.UpdatedAt.Unix(),
		})
	}

	return &pb.GetUserListResponse{
		Users:    users,
		Total:    resp.Total,
		Page:     int32(resp.Page),
		PageSize: int32(resp.PageSize),
	}, nil
}
