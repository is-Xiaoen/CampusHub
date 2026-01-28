# 部署配置文件

此目录存放部署到服务器的配置文件。

## 使用方法

1. 从 `app/*/rpc/etc/*.yaml.example` 复制模板到此目录
2. 根据实际环境修改配置
3. 重命名为对应服务名

```bash
# 示例
cp ../../app/gateway/api/etc/gateway.yaml.example ./gateway.yaml
cp ../../app/user/rpc/etc/user.yaml.example ./user.yaml
cp ../../app/activity/rpc/etc/activity.yaml.example ./activity.yaml
cp ../../app/chat/rpc/etc/chat.yaml.example ./chat.yaml
cp ../../app/demo/rpc/etc/demo.yaml.example ./demo.yaml

# 然后编辑各文件，填入实际配置
```

## 当前环境配置

| 配置项 | 值 |
|--------|-----|
| MySQL | 192.168.10.4:3308 |
| Redis | 192.168.10.4:6379 |
| Etcd | 192.168.10.4:2379 |
| 应用服务器 | 192.168.10.9 |

## 注意事项

- 此目录下的 yaml 文件包含敏感信息
- 已在 .gitignore 中排除
- 请勿提交到公共仓库
