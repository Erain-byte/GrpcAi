package server

import (
	"context"

	grpcclient "gateway/internal/grpc"
	pbAi "github.com/Erain-byte/GrpcAi/proto/ai"
	"gateway/internal/svc"
)

// AiForwarder AI 服务转发器
type AiForwarder struct {
	pbAi.UnimplementedAiServiceServer
	base *BaseForwarder[pbAi.AiServiceClient]
}

// NewAiForwarder 创建 AI 服务转发器
func NewAiForwarder(svcCtx *svc.ServiceContext, clientMgr *grpcclient.ClientManager) *AiForwarder {
	return &AiForwarder{
		base: NewBaseForwarder(
			svcCtx,
			clientMgr,
			"ai-service",
			pbAi.NewAiServiceClient,
		),
	}
}

// Chat AI 聊天
func (f *AiForwarder) Chat(ctx context.Context, req *pbAi.ChatRequest) (*pbAi.ChatResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Chat(ctx, req)
}

// StreamChat AI 流式聊天
func (f *AiForwarder) StreamChat(req *pbAi.StreamChatRequest, stream pbAi.AiService_StreamChatServer) error {
	client, err := f.base.GetClient(stream.Context())
	if err != nil {
		return err
	}
	
	respStream, err := client.StreamChat(stream.Context(), req)
	if err != nil {
		return err
	}
	
	for {
		resp, err := respStream.Recv()
		if err != nil {
			return err
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		if resp.GetDone() {
			return nil
		}
	}
}

// GetChatHistory 获取聊天历史
func (f *AiForwarder) GetChatHistory(ctx context.Context, req *pbAi.GetChatHistoryRequest) (*pbAi.GetChatHistoryResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetChatHistory(ctx, req)
}

// GetChatList 获取聊天列表
func (f *AiForwarder) GetChatList(ctx context.Context, req *pbAi.GetChatListRequest) (*pbAi.GetChatListResponse, error) {
	client, err := f.base.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetChatList(ctx, req)
}
