package redis

import "fmt"

// 错误类型
var (
	ErrInvalidConfig = func(msg string) error {
		return fmt.Errorf("无效的redis配置: %s", msg)
	}
	ErrConnectionFailed = func(err error) error {
		return fmt.Errorf("redis连接失败: %w", err)
	}
	ErrPublishFailed = func(err error) error {
		return fmt.Errorf("redis发布失败: %w", err)
	}
	ErrSubscribeFailed = func(err error) error {
		return fmt.Errorf("redis订阅失败: %w", err)
	}
	ErrConsumerGroupExists = func(topic, group string) error {
		return fmt.Errorf("消费者组 %s 已存在于主题 %s", group, topic)
	}
)
