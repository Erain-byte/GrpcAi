# Proto

统一的 gRPC Proto 定义仓库，所有微服务共享。

## 目录结构

```
proto/
├── user/user.proto    # 用户服务
├── admin/admin.proto  # 管理员服务
├── ai/ai.proto        # AI服务
├── go.mod
└── gen.sh             # 生成脚本
```

## 使用方式

各服务通过 Go Module 引用：

```bash
go get github.com/yourname/proto@latest
```

## 生成代码

```bash
# 安装依赖
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

# 执行生成
bash gen.sh
```

## 推送更新

1. 修改 .proto 文件
2. 执行 gen.sh 生成代码
3. 提交并推送到 GitHub
4. 各服务 `go get github.com/yourname/proto@latest` 更新依赖
