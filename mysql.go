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
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New 根据给定的选项创建一个新的 GORM 数据库实例.
func New(opts *Options) (*gorm.DB, error) {
	// 构建 DSN (Data Source Name)
	dsn := fmt.Sprintf(`%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=%t&loc=%s`,
		opts.Username,
		opts.Password,
		opts.Host,
		opts.Database,
		true,    // parseTime=true 才能将 MySQL 的 DATETIME/TIMESTAMP 正确解析为 Go 的 time.Time
		"Local") // 使用本地时区

	return newDB(dsn, opts)
}

// newDB 内部函数，用于创建数据库连接
func newDB(dsn string, opts *Options) (*gorm.DB, error) {
	// 确保 Logger 不为 nil，否则 GORM 可能会使用默认的 logger
	var gormLogger logger.Interface
	if opts.Logger != nil {
		gormLogger = opts.Logger
	} else {
		// 如果未提供自定义 logger，可以创建一个默认的 logger
		gormLogger = logger.Default.LogMode(opts.LogLevel)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	if opts.MaxOpenConnections > 0 {
		sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)
	}
	if opts.MaxConnectionLifeTime > 0 {
		sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)
	}
	if opts.MaxIdleConnections > 0 {
		sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)
	}

	// 如果启用了追踪，则注册 GormTracePlugin
	if opts.EnableTrace {
		if err := db.Use(NewGormTracePlugin(true)); err != nil {
			return nil, fmt.Errorf("failed to register trace plugin: %w", err)
		}
	}

	return db, nil
}
