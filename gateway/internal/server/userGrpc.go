package server

import (
	"context"

	grpcclient "gateway/internal/grpc"
	"gateway/internal/logger"
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
	// ⭐ 使用带 Trace ID 的日志器
	log := logger.NewContextSugaredLogger(ctx)
	
	client, err := f.base.GetClient(ctx)
	if err != nil {
		log.Errorf("Failed to get user service client: %v", err)
		return nil, err
	}
	
	log.Infof("Processing user registration [username=%s]", req.Username)
	resp, err := client.Register(ctx, req)
	if err != nil {
		log.Errorf("User registration failed: %v", err)
		return nil, err
	}
	
	log.Info("User registration successful")
	return resp, nil
}

// Login 用户登录
func (f *UserForwarder) Login(ctx context.Context, req *pbUser.LoginRequest) (*pbUser.LoginResponse, error) {
	log := logger.NewContextSugaredLogger(ctx)
	
	client, err := f.base.GetClient(ctx)
	if err != nil {
		log.Errorf("Failed to get user service client: %v", err)
		return nil, err
	}
	
	log.Infof("Processing user login [username=%s]", req.Username)
	resp, err := client.Login(ctx, req)
	if err != nil {
		log.Warnf("User login failed: %v", err)
		return nil, err
	}
	
	log.Info("User login successful")
	return resp, nil
}

// Logout 用户登出
func (f *UserForwarder) Logout(ctx context.Context, req *pbUser.LogoutRequest) (*pbUser.LogoutResponse, error) {
	log := logger.NewContextSugaredLogger(ctx)
	
	client, err := f.base.GetClient(ctx)
	if err != nil {
		log.Errorf("Failed to get user service client: %v", err)
		return nil, err
	}
	
	log.Info("Processing user logout")
	resp, err := client.Logout(ctx, req)
	if err != nil {
		log.Errorf("User logout failed: %v", err)
		return nil, err
	}
	
	log.Info("User logout successful")
	return resp, nil
}

// GetUserList 获取用户列表
func (f *UserForwarder) GetUserList(ctx context.Context, req *pbUser.GetUserListRequest) (*pbUser.GetUserListResponse, error) {
	log := logger.NewContextSugaredLogger(ctx)
	
	client, err := f.base.GetClient(ctx)
	if err != nil {
		log.Errorf("Failed to get user service client: %v", err)
		return nil, err
	}
	
	log.Debug("Fetching user list")
	resp, err := client.GetUserList(ctx, req)
	if err != nil {
		log.Errorf("Failed to get user list: %v", err)
		return nil, err
	}
	
	log.Debugf("User list retrieved [count=%d]", len(resp.Users))
	return resp, nil
}
