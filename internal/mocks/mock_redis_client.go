package mocks

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type MockRedisClient struct {
	redis.Cmdable

	GetFunc        func(ctx context.Context, key string) *redis.StringCmd
	TxPipelineFunc func() redis.Pipeliner
	DelFunc        func(ctx context.Context, keys ...string) *redis.IntCmd
	SRemFunc       func(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return m.GetFunc(ctx, key)
}

func (m *MockRedisClient) TxPipeline() redis.Pipeliner {
	return m.TxPipelineFunc()
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return m.DelFunc(ctx, keys...)
}

func (m *MockRedisClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return m.SRemFunc(ctx, key, members...)
}

type MockTxPipeline struct {
	redis.Pipeliner

	SetNXFunc func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	SetFunc   func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SAddFunc  func(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ExecFunc  func(ctx context.Context) ([]redis.Cmder, error)
}

func (p *MockTxPipeline) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return p.SetNXFunc(ctx, key, value, expiration)
}

func (p *MockTxPipeline) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return p.SetFunc(ctx, key, value, expiration)
}

func (p *MockTxPipeline) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return p.SAddFunc(ctx, key, members...)
}

func (p *MockTxPipeline) Exec(ctx context.Context) ([]redis.Cmder, error) {
	return p.ExecFunc(ctx)
}
