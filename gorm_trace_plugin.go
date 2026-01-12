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
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/go-anyway/framework-log"
	"github.com/go-anyway/framework-metrics"
	pkgtrace "github.com/go-anyway/framework-trace"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	callBackBeforeName = "core:before"
	callBackAfterName  = "core:after"
	startTime          = "_start_time"
	spanKey            = "_span"
)

// GormTracePlugin 定义了一个 GORM 插件，用于追踪 SQL 查询的执行时间（支持 OpenTelemetry）
type GormTracePlugin struct {
	enableTrace bool // 是否启用 OpenTelemetry 追踪
}

// NewGormTracePlugin 创建新的 GORM 追踪插件
func NewGormTracePlugin(enableTrace bool) *GormTracePlugin {
	return &GormTracePlugin{
		enableTrace: enableTrace,
	}
}

// Name 返回追踪插件的名称
func (op *GormTracePlugin) Name() string {
	return "GormTracePlugin"
}

// Initialize 初始化追踪插件，注册 GORM 回调
func (op *GormTracePlugin) Initialize(db *gorm.DB) (err error) {
	// 在操作开始前注册回调
	_ = db.Callback().Create().Before("gorm:before_create").Register(callBackBeforeName, op.before)
	_ = db.Callback().Query().Before("gorm:query").Register(callBackBeforeName, op.before)
	_ = db.Callback().Delete().Before("gorm:before_delete").Register(callBackBeforeName, op.before)
	_ = db.Callback().Update().Before("gorm:setup_reflect_value").Register(callBackBeforeName, op.before)
	_ = db.Callback().Row().Before("gorm:row").Register(callBackBeforeName, op.before)
	_ = db.Callback().Raw().Before("gorm:raw").Register(callBackBeforeName, op.before)

	// 在操作结束后注册回调
	_ = db.Callback().Create().After("gorm:after_create").Register(callBackAfterName, op.after)
	_ = db.Callback().Query().After("gorm:after_query").Register(callBackAfterName, op.after)
	_ = db.Callback().Delete().After("gorm:after_delete").Register(callBackAfterName, op.after)
	_ = db.Callback().Update().After("gorm:after_update").Register(callBackAfterName, op.after)
	_ = db.Callback().Row().After("gorm:row").Register(callBackAfterName, op.after)
	_ = db.Callback().Raw().After("gorm:raw").Register(callBackAfterName, op.after)

	return
}

// 确保 GormTracePlugin 实现了 gorm.Plugin 接口
var _ gorm.Plugin = &GormTracePlugin{}

// before 是 GORM 操作开始前的回调函数，记录当前时间并创建追踪 span
func (op *GormTracePlugin) before(db *gorm.DB) {
	// 记录开始时间
	db.InstanceSet(startTime, time.Now())

	// 如果启用了追踪，创建 OpenTelemetry span
	if op.enableTrace {
		var ctx context.Context
		if db.Statement != nil && db.Statement.Context != nil {
			ctx = db.Statement.Context
		} else {
			ctx = context.Background()
		}

		// 确定操作类型
		operation := getOperationType(db)
		spanName := "gorm." + operation

		// 创建 span
		ctx, span := pkgtrace.StartSpan(ctx, spanName,
			trace.WithAttributes(
				attribute.String("db.system", "sql"), // 通用 SQL 数据库
				attribute.String("db.operation", operation),
			),
		)

		// 保存 span 到实例中
		db.InstanceSet(spanKey, span)
		// 更新 context（如果 Statement 存在）
		if db.Statement != nil {
			db.Statement.Context = ctx
		}
	}
}

// after 是 GORM 操作结束后的回调函数，计算并记录 SQL 执行时间
func (op *GormTracePlugin) after(db *gorm.DB) {
	_ts, isExist := db.InstanceGet(startTime)
	if !isExist {
		return
	}

	ts, ok := _ts.(time.Time)
	if !ok {
		return
	}

	duration := time.Since(ts)
	durationSeconds := duration.Seconds()

	// 确定操作类型
	operation := getOperationType(db)

	// 确定状态（成功或失败）
	status := "success"
	if db.Error != nil {
		status = "error"
	}

	// 获取完整的 SQL 语句（带实际参数值）
	sql := getFullSQL(db)

	// 如果启用了追踪，更新 span
	if op.enableTrace {
		if spanVal, exists := db.InstanceGet(spanKey); exists {
			if span, ok := spanVal.(trace.Span); ok && span != nil {
				// 设置 span 属性
				span.SetAttributes(
					attribute.String("db.statement", sql),
					attribute.String("db.operation", operation),
					attribute.Float64("db.duration_ms", float64(duration.Milliseconds())),
				)

				// 设置状态
				if db.Error != nil {
					span.SetStatus(codes.Error, db.Error.Error())
					span.RecordError(db.Error)
				} else {
					span.SetStatus(codes.Ok, "")
				}

				// 结束 span
				span.End()
			}
		}
	}

	// 记录日志
	log.FromContext(db.Statement.Context).Info(
		"SQL cost time",
		zap.Float64("cost_ms", float64(duration.Microseconds())/1000.0),
		zap.String("sql", sql),
		zap.String("operation", operation),
		zap.String("status", status),
	)

	// 记录 Prometheus 指标（仅在启用时）
	if metrics.IsEnabled() {
		metrics.DatabaseQueryTotal.WithLabelValues(operation, status).Inc()
		metrics.DatabaseQueryDuration.WithLabelValues(operation).Observe(durationSeconds)
	}
}

// getOperationType 根据 GORM 的 Statement 确定操作类型
func getOperationType(db *gorm.DB) string {
	if db.Statement == nil {
		return "unknown"
	}

	// 首先尝试从 SQL 语句判断（在 after 回调中 SQL 已经构建）
	sql := db.Statement.SQL.String()
	if len(sql) > 0 {
		// 转换为大写并提取前几个字符用于判断
		sqlUpper := strings.ToUpper(strings.TrimSpace(sql))
		if len(sqlUpper) >= 6 {
			// 根据 SQL 语句开头判断操作类型
			switch {
			case strings.HasPrefix(sqlUpper, "SELECT"):
				return "select"
			case strings.HasPrefix(sqlUpper, "INSERT"):
				return "insert"
			case strings.HasPrefix(sqlUpper, "UPDATE"):
				return "update"
			case strings.HasPrefix(sqlUpper, "DELETE"):
				return "delete"
			}
		}
	}

	// 如果 SQL 为空（在 before 回调中），尝试通过 Statement 的其他字段判断
	// 检查是否有 Model 或 Table 信息
	if db.Statement.Model != nil || db.Statement.Table != "" {
		// 通过检查 Statement 的某些特征来推断操作类型
		// 注意：这是一个启发式方法，可能不够准确
		// 在 before 回调中，我们使用 "query" 作为默认值
		// 在 after 回调中，SQL 已经构建，会使用上面的 SQL 判断逻辑
		return "query"
	}

	// 如果无法判断，返回通用类型
	return "other"
}

// getFullSQL 获取完整的 SQL 语句（带实际参数值）
func getFullSQL(db *gorm.DB) string {
	if db.Statement == nil {
		return ""
	}

	// 获取参数化的 SQL
	sql := db.Statement.SQL.String()
	if sql == "" {
		return ""
	}

	// 如果没有参数，直接返回 SQL
	if len(db.Statement.Vars) == 0 {
		return sql
	}

	// 手动构建完整的 SQL：将参数值替换到 SQL 中
	// 注意：这是一个简化的实现，对于复杂情况可能不够准确
	// 但对于大多数情况（字符串、数字、时间）应该足够
	result := sql
	paramIndex := 0
	for paramIndex < len(db.Statement.Vars) {
		// 查找下一个 ? 占位符
		pos := strings.Index(result, "?")
		if pos == -1 {
			break
		}

		// 获取参数值并格式化为字符串
		var paramStr string
		param := db.Statement.Vars[paramIndex]
		switch v := param.(type) {
		case string:
			paramStr = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		case []byte:
			paramStr = fmt.Sprintf("'%s'", strings.ReplaceAll(string(v), "'", "''"))
		case time.Time:
			paramStr = fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05"))
		case nil:
			paramStr = "NULL"
		default:
			// 对于数字和其他类型，直接转换为字符串
			paramStr = fmt.Sprintf("%v", v)
		}

		// 替换占位符
		result = result[:pos] + paramStr + result[pos+1:]
		paramIndex++
	}

	return result
}
