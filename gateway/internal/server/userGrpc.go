package server

import (
	"context"

	grpcclient "gateway/internal/grpc"
	pbUser "github.com/Erain-byte/GrpcAi/proto/user"
	"gateway/internal/svc"
)

// UserForwarder 用户服务转发器
type UserForwarder struct {
	pbUser.UnimplementedUserServiceServer
	base *BaseForwarder[pbUser.UserServiceClient]
}

// NewUserForwarder 创建用户服务转发器
func NewUserForwarder(svcCtx *svc.ServiceContext, clientMgr *grpcclient.ClientManager) *UserForwarder {
	return &UserForwarder{
		base: NewBaseForwarder(
			svcCtx,
			clientMgr,
			"user-service",
			pbUser.NewUserServiceClient,
		),
	}
}

// Register 用户注册
func (f *UserForwarder) Register(ctx context.Context, req *pbUser.RegisterRequest) (*pbUser.RegisterResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Register(ctx, req)
}

// Login 用户登录
func (f *UserForwarder) Login(ctx context.Context, req *pbUser.LoginRequest) (*pbUser.LoginResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Login(ctx, req)
}

// Logout 用户登出
func (f *UserForwarder) Logout(ctx context.Context, req *pbUser.LogoutRequest) (*pbUser.LogoutResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Logout(ctx, req)
}

// GetUserList 获取用户列表
func (f *UserForwarder) GetUserList(ctx context.Context, req *pbUser.GetUserListRequest) (*pbUser.GetUserListResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetUserList(ctx, req)
}
