// Copyright 2025 zampo.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// @contact  zampo3380@gmail.com

package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedis 根据给定的选项创建一个新的 Redis 客户端实例
func NewRedis(opts *RedisOptions) (*redis.Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("redis options cannot be nil")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:            opts.Addr,
		Password:        opts.Password,
		DB:              opts.DB,
		PoolSize:        opts.PoolSize,
		MinIdleConns:    opts.MinIdleConns,
		DialTimeout:     opts.DialTimeout,
		ReadTimeout:     opts.ReadTimeout,
		WriteTimeout:    opts.WriteTimeout,
		ConnMaxIdleTime: opts.IdleTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), opts.DialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 如果启用了追踪，则添加追踪 Hook
	if opts.EnableTrace {
		addTraceHook(rdb, opts.EnableTrace)
	}

	return rdb, nil
}
