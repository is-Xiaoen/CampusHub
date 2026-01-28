#!/bin/bash
# Proto 文件生成脚本

set -e

PROTO_DIR="./app"

echo "=== 生成 Proto 文件 ==="

# User service
if [ -f "$PROTO_DIR/user/rpc/user.proto" ]; then
    echo "生成 User Proto..."
    protoc --go_out=. --go-grpc_out=. $PROTO_DIR/user/rpc/user.proto
fi

# Activity service
if [ -f "$PROTO_DIR/activity/rpc/activity.proto" ]; then
    echo "生成 Activity Proto..."
    protoc --go_out=. --go-grpc_out=. $PROTO_DIR/activity/rpc/activity.proto
fi

# Chat service
if [ -f "$PROTO_DIR/chat/rpc/chat.proto" ]; then
    echo "生成 Chat Proto..."
    protoc --go_out=. --go-grpc_out=. $PROTO_DIR/chat/rpc/chat.proto
fi

echo "=== Proto 生成完成 ==="
