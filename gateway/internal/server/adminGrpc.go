package server

import (
	"context"

	grpcclient "gateway/internal/grpc"
	pbAdmin "github.com/Erain-byte/GrpcAi/proto/admin"
	"gateway/internal/svc"
)

// AdminForwarder 管理服务转发器
type AdminForwarder struct {
	pbAdmin.UnimplementedAdminServiceServer
	base *BaseForwarder[pbAdmin.AdminServiceClient]
}

// NewAdminForwarder 创建管理服务转发器
func NewAdminForwarder(svcCtx *svc.ServiceContext, clientMgr *grpcclient.ClientManager) *AdminForwarder {
	return &AdminForwarder{
		base: NewBaseForwarder(
			svcCtx,
			clientMgr,
			"admin-service",
			pbAdmin.NewAdminServiceClient,
		),
	}
}

// Login 管理员登录
func (f *AdminForwarder) Login(ctx context.Context, req *pbAdmin.AdminLoginRequest) (*pbAdmin.AdminLoginResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Login(ctx, req)
}

// Logout 管理员登出
func (f *AdminForwarder) Logout(ctx context.Context, req *pbAdmin.LogoutRequest) (*pbAdmin.LogoutResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Logout(ctx, req)
}

// GetAdminInfo 获取管理员信息
func (f *AdminForwarder) GetAdminInfo(ctx context.Context, req *pbAdmin.GetAdminInfoRequest) (*pbAdmin.GetAdminInfoResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetAdminInfo(ctx, req)
}

// CreateAdmin 创建管理员
func (f *AdminForwarder) CreateAdmin(ctx context.Context, req *pbAdmin.CreateAdminRequest) (*pbAdmin.CreateAdminResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.CreateAdmin(ctx, req)
}

// GetAdminList 获取管理员列表
func (f *AdminForwarder) GetAdminList(ctx context.Context, req *pbAdmin.GetAdminListRequest) (*pbAdmin.GetAdminListResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetAdminList(ctx, req)
}
