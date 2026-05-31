package server

import (
	"log"

	grpcclient "gateway/internal/grpc"
	pbAdmin "github.com/Erain-byte/GrpcAi/proto/admin"
	pbAi "github.com/Erain-byte/GrpcAi/proto/ai"
	pbUser "github.com/Erain-byte/GrpcAi/proto/user"
	"gateway/internal/svc"

	"google.golang.org/grpc"
)

// RegisterAllGRPCServices 注册所有 gRPC 服务到网关
func RegisterAllGRPCServices(grpcSrv *grpc.Server, svcCtx *svc.ServiceContext, grpcClients *grpcclient.ClientManager) {
	// 创建各个服务的转发器
	userForwarder := NewUserForwarder(svcCtx, grpcClients)
	adminForwarder := NewAdminForwarder(svcCtx, grpcClients)
	aiForwarder := NewAiForwarder(svcCtx, grpcClients)

	// 注册 User 服务
	pbUser.RegisterUserServiceServer(grpcSrv, userForwarder)
	log.Println("User service registered")

	// 注册 Admin 服务
	pbAdmin.RegisterAdminServiceServer(grpcSrv, adminForwarder)
	log.Println("Admin service registered")

	// 注册 AI 服务
	pbAi.RegisterAiServiceServer(grpcSrv, aiForwarder)
	log.Println("AI service registered")

	log.Println("All gRPC services registered to gateway")
}
