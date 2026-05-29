#!/bin/bash

# Proto 代码生成脚本

set -e

PROTO_DIR=$(dirname "$0")
THIRD_PARTY="${PROTO_DIR}/third_party"

# 确保 third_party 目录存在
mkdir -p "${THIRD_PARTY}/google/api"

# 下载 google/api/annotations.proto 如果不存在
if [ ! -f "${THIRD_PARTY}/google/api/annotations.proto" ]; then
    echo "Downloading google/api/annotations.proto..."
    curl -sL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto -o "${THIRD_PARTY}/google/api/annotations.proto"
fi

if [ ! -f "${THIRD_PARTY}/google/api/http.proto" ]; then
    echo "Downloading google/api/http.proto..."
    curl -sL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto -o "${THIRD_PARTY}/google/api/http.proto"
fi

# 生成函数
generate() {
    local name=$1
    local proto_file="${PROTO_DIR}/${name}/${name}.proto"

    echo "Generating ${name}..."

    # 生成 Go protobuf 代码
    protoc         --proto_path="${PROTO_DIR}"         --proto_path="${THIRD_PARTY}"         --go_out="${PROTO_DIR}"         --go_opt=paths=source_relative         --go-grpc_out="${PROTO_DIR}"         --go-grpc_opt=paths=source_relative         "${proto_file}"

    echo "  ✓ ${name} pb generated"
}

# 生成各服务代码
generate "user"
generate "admin"
generate "ai"

echo ""
echo "All proto files generated successfully!"
