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
	"time"

	"github.com/go-anyway/framework-log"
	"github.com/go-anyway/framework-metrics"
	pkgtrace "github.com/go-anyway/framework-trace"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// traceRedisHook 实现 redis.Hook 接口用于追踪命令（支持 OpenTelemetry）
type traceRedisHook struct {
	enableTrace bool // 是否启用 OpenTelemetry 追踪
}

// newTraceRedisHook 创建新的 Redis 追踪 Hook
func newTraceRedisHook(enableTrace bool) *traceRedisHook {
	return &traceRedisHook{
		enableTrace: enableTrace,
	}
}

// DialHook 在建立连接时调用
func (h *traceRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook 在处理命令时调用
func (h *traceRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// 如果启用了追踪，创建 OpenTelemetry span
		var span trace.Span
		if h.enableTrace {
			operation := cmd.Name()
			if operation == "" {
				operation = "unknown"
			}
			ctx, span = pkgtrace.StartSpan(ctx, "redis."+operation,
				trace.WithAttributes(
					attribute.String("db.system", "redis"),
					attribute.String("db.operation", operation),
				),
			)
			defer span.End()
		}

		start := time.Now()
		err := next(ctx, cmd)
		duration := time.Since(start)
		durationSeconds := duration.Seconds()

		// 确定操作类型和状态
		operation := cmd.Name()
		if operation == "" {
			operation = "unknown"
		}
		status := "success"
		if err != nil {
			status = "error"
		}

		// 如果启用了追踪，更新 span
		if h.enableTrace && span != nil {
			span.SetAttributes(
				attribute.String("db.statement", cmd.String()),
				attribute.Float64("db.duration_ms", float64(duration.Milliseconds())),
			)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
			} else {
				span.SetStatus(codes.Ok, "")
			}
		}

		// 记录日志
		log.FromContext(ctx).Info(
			"Redis command success",
			zap.String("operation", operation),
			zap.String("cmd", cmd.String()),
			zap.Duration("duration", duration),
			zap.String("status", status),
		)

		// 记录 Prometheus 指标（仅在启用时）
		if metrics.IsEnabled() {
			metrics.RedisOperationTotal.WithLabelValues(operation, status).Inc()
			metrics.RedisOperationDuration.WithLabelValues(operation).Observe(durationSeconds)
		}

		return err
	}
}

// ProcessPipelineHook 在处理管道命令时调用
func (h *traceRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		// 如果启用了追踪，创建 OpenTelemetry span
		var span trace.Span
		if h.enableTrace {
			ctx, span = pkgtrace.StartSpan(ctx, "redis.pipeline",
				trace.WithAttributes(
					attribute.String("db.system", "redis"),
					attribute.String("db.operation", "pipeline"),
					attribute.Int("db.command_count", len(cmds)),
				),
			)
			defer span.End()
		}

		start := time.Now()
		err := next(ctx, cmds)
		duration := time.Since(start)
		durationSeconds := duration.Seconds()

		// 确定状态
		status := "success"
		if err != nil {
			status = "error"
		}

		// 如果启用了追踪，更新 span
		if h.enableTrace && span != nil {
			span.SetAttributes(
				attribute.Float64("db.duration_ms", float64(duration.Milliseconds())),
			)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
			} else {
				span.SetStatus(codes.Ok, "")
			}
		}

		// 记录日志
		log.FromContext(ctx).Info(
			"Redis pipeline success",
			zap.Int("cmd_count", len(cmds)),
			zap.Duration("duration", duration),
			zap.String("status", status),
		)

		// 记录 Prometheus 指标（仅在启用时，管道操作使用 "pipeline" 作为操作类型）
		if metrics.IsEnabled() {
			metrics.RedisOperationTotal.WithLabelValues("pipeline", status).Inc()
			metrics.RedisOperationDuration.WithLabelValues("pipeline").Observe(durationSeconds)
		}

		return err
	}
}

// addTraceHook 为 Redis 客户端添加追踪 Hook
func addTraceHook(client *redis.Client, enableTrace bool) {
	hook := newTraceRedisHook(enableTrace)
	client.AddHook(hook)
}
