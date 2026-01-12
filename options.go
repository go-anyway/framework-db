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
	"time"

	pkgConfig "github.com/go-anyway/framework-config"

	"gorm.io/gorm/logger"
)

// MySQLConfig MySQL 数据库配置结构体（用于从配置文件创建）
type MySQLConfig struct {
	Enabled        bool               `yaml:"enabled" env:"MYSQL_ENABLED" default:"true"`
	Host           string             `yaml:"host" env:"MYSQL_HOST" default:"localhost"`
	Port           int                `yaml:"port" env:"MYSQL_PORT" default:"3306"`
	Database       string             `yaml:"database" env:"MYSQL_DATABASE" required:"true"`
	Username       string             `yaml:"username" env:"MYSQL_USERNAME" required:"true"`
	Password       string             `yaml:"password" env:"MYSQL_PASSWORD" required:"true"`
	MaxConnections int                `yaml:"max_connections" env:"MYSQL_MAX_CONNECTIONS" default:"100"`
	Timeout        pkgConfig.Duration `yaml:"timeout" env:"MYSQL_TIMEOUT" default:"30s"`
	Charset        string             `yaml:"charset" env:"MYSQL_CHARSET" default:"utf8mb4"`
	ParseTime      bool               `yaml:"parse_time" env:"MYSQL_PARSE_TIME" default:"true"`
	Loc            string             `yaml:"loc" env:"MYSQL_LOC" default:"Local"`
	EnableTrace    bool               `yaml:"enable_trace" env:"MYSQL_ENABLE_TRACE" default:"true"`
}

// Validate 验证 MySQL 配置
func (c *MySQLConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("mysql config cannot be nil")
	}
	if !c.Enabled {
		return nil // 如果未启用，不需要验证
	}
	if c.Database == "" {
		return fmt.Errorf("mysql database is required")
	}
	if c.Username == "" {
		return fmt.Errorf("mysql username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("mysql password is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("mysql port must be between 1 and 65535, got %d", c.Port)
	}
	if c.MaxConnections < 1 {
		return fmt.Errorf("mysql max_connections must be greater than 0, got %d", c.MaxConnections)
	}
	return nil
}

// ToOptions 转换为 Options
func (c *MySQLConfig) ToOptions() (*Options, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, fmt.Errorf("mysql is not enabled")
	}

	timeout := c.Timeout.Duration()
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Options{
		Host:                  fmt.Sprintf("%s:%d", c.Host, c.Port),
		Username:              c.Username,
		Password:              c.Password,
		Database:              c.Database,
		MaxIdleConnections:    c.MaxConnections / 10, // 默认空闲连接数为最大连接数的 10%
		MaxOpenConnections:    c.MaxConnections,
		MaxConnectionLifeTime: timeout,
		LogLevel:              logger.Info,
		EnableTrace:           c.EnableTrace,
	}, nil
}

// DSN 返回 MySQL 数据源名称
func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.Charset, c.ParseTime, c.Loc)
}

// PostgreSQLConfig PostgreSQL 配置结构体（用于从配置文件创建）
type PostgreSQLConfig struct {
	Enabled        bool               `yaml:"enabled" env:"POSTGRESQL_ENABLED" default:"true"`
	Host           string             `yaml:"host" env:"POSTGRESQL_HOST" default:"localhost"`
	Port           int                `yaml:"port" env:"POSTGRESQL_PORT" default:"5432"`
	Database       string             `yaml:"database" env:"POSTGRESQL_DATABASE" required:"true"`
	Username       string             `yaml:"username" env:"POSTGRESQL_USERNAME" required:"true"`
	Password       string             `yaml:"password" env:"POSTGRESQL_PASSWORD" required:"true"`
	SSLMode        string             `yaml:"ssl_mode" env:"POSTGRESQL_SSL_MODE" default:"disable"`
	MaxConnections int                `yaml:"max_connections" env:"POSTGRESQL_MAX_CONNECTIONS" default:"100"`
	Timeout        pkgConfig.Duration `yaml:"timeout" env:"POSTGRESQL_TIMEOUT" default:"30s"`
	EnableTrace    bool               `yaml:"enable_trace" env:"POSTGRESQL_ENABLE_TRACE" default:"true"`
}

// Validate 验证 PostgreSQL 配置
func (c *PostgreSQLConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("postgresql config cannot be nil")
	}
	if !c.Enabled {
		return nil // 如果未启用，不需要验证
	}
	if c.Database == "" {
		return fmt.Errorf("postgresql database is required")
	}
	if c.Username == "" {
		return fmt.Errorf("postgresql username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("postgresql password is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("postgresql port must be between 1 and 65535, got %d", c.Port)
	}
	if c.MaxConnections < 1 {
		return fmt.Errorf("postgresql max_connections must be greater than 0, got %d", c.MaxConnections)
	}
	// 验证 SSLMode 的有效值
	validSSLModes := map[string]bool{
		"disable": true, "allow": true, "prefer": true, "require": true,
		"verify-ca": true, "verify-full": true,
	}
	if c.SSLMode != "" && !validSSLModes[c.SSLMode] {
		return fmt.Errorf("postgresql ssl_mode must be one of: disable, allow, prefer, require, verify-ca, verify-full, got %s", c.SSLMode)
	}
	return nil
}

// ToOptions 转换为 PostgreSQLOptions
func (c *PostgreSQLConfig) ToOptions() (*PostgreSQLOptions, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, fmt.Errorf("postgresql is not enabled")
	}

	timeout := c.Timeout.Duration()
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &PostgreSQLOptions{
		Host:                  c.Host,
		Port:                  c.Port,
		Username:              c.Username,
		Password:              c.Password,
		Database:              c.Database,
		SSLMode:               c.SSLMode,
		MaxIdleConnections:    c.MaxConnections / 10, // 默认空闲连接数为最大连接数的 10%
		MaxOpenConnections:    c.MaxConnections,
		MaxConnectionLifeTime: timeout,
		LogLevel:              logger.Info,
		EnableTrace:           c.EnableTrace,
	}, nil
}

// TimeoutDuration 返回 time.Duration 类型的 Timeout
func (c *PostgreSQLConfig) TimeoutDuration() time.Duration {
	return c.Timeout.Duration()
}

// RedisConfig Redis 配置结构体（用于从配置文件创建）
type RedisConfig struct {
	Enabled      bool               `yaml:"enabled" env:"REDIS_ENABLED" default:"true"`
	Host         string             `yaml:"host" env:"REDIS_HOST" default:"localhost"`
	Port         int                `yaml:"port" env:"REDIS_PORT" default:"6379"`
	Password     string             `yaml:"password" env:"REDIS_PASSWORD"`
	DB           int                `yaml:"db" env:"REDIS_DB" default:"0"`
	PoolSize     int                `yaml:"pool_size" env:"REDIS_POOL_SIZE" default:"20"`
	MinIdleConns int                `yaml:"min_idle_conns" env:"REDIS_MIN_IDLE_CONNS" default:"5"`
	DialTimeout  pkgConfig.Duration `yaml:"dial_timeout" env:"REDIS_DIAL_TIMEOUT" default:"5s"`
	ReadTimeout  pkgConfig.Duration `yaml:"read_timeout" env:"REDIS_READ_TIMEOUT" default:"3s"`
	WriteTimeout pkgConfig.Duration `yaml:"write_timeout" env:"REDIS_WRITE_TIMEOUT" default:"3s"`
	IdleTimeout  pkgConfig.Duration `yaml:"idle_timeout" env:"REDIS_IDLE_TIMEOUT" default:"5m"`
	EnableTrace  bool               `yaml:"enable_trace" env:"REDIS_ENABLE_TRACE" default:"true"`
}

// Validate 验证 Redis 配置
func (c *RedisConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("redis config cannot be nil")
	}
	if !c.Enabled {
		return nil // 如果未启用，不需要验证
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("redis port must be between 1 and 65535, got %d", c.Port)
	}
	if c.PoolSize < 1 {
		return fmt.Errorf("redis pool_size must be greater than 0, got %d", c.PoolSize)
	}
	if c.MinIdleConns < 0 {
		return fmt.Errorf("redis min_idle_conns must be non-negative, got %d", c.MinIdleConns)
	}
	if c.DB < 0 || c.DB > 15 {
		return fmt.Errorf("redis db must be between 0 and 15, got %d", c.DB)
	}
	return nil
}

// ToOptions 转换为 RedisOptions
func (c *RedisConfig) ToOptions() (*RedisOptions, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, fmt.Errorf("redis is not enabled")
	}

	dialTimeout := c.DialTimeout.Duration()
	if dialTimeout == 0 {
		dialTimeout = 5 * time.Second
	}
	readTimeout := c.ReadTimeout.Duration()
	if readTimeout == 0 {
		readTimeout = 3 * time.Second
	}
	writeTimeout := c.WriteTimeout.Duration()
	if writeTimeout == 0 {
		writeTimeout = 3 * time.Second
	}
	idleTimeout := c.IdleTimeout.Duration()
	if idleTimeout == 0 {
		idleTimeout = 5 * time.Minute
	}

	return &RedisOptions{
		Addr:         fmt.Sprintf("%s:%d", c.Host, c.Port),
		Password:     c.Password,
		DB:           c.DB,
		PoolSize:     c.PoolSize,
		MinIdleConns: c.MinIdleConns,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		EnableTrace:  c.EnableTrace,
	}, nil
}

// Addr 返回 Redis 地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// DialTimeoutDuration 返回 time.Duration 类型的 DialTimeout
func (c *RedisConfig) DialTimeoutDuration() time.Duration {
	return c.DialTimeout.Duration()
}

// ReadTimeoutDuration 返回 time.Duration 类型的 ReadTimeout
func (c *RedisConfig) ReadTimeoutDuration() time.Duration {
	return c.ReadTimeout.Duration()
}

// WriteTimeoutDuration 返回 time.Duration 类型的 WriteTimeout
func (c *RedisConfig) WriteTimeoutDuration() time.Duration {
	return c.WriteTimeout.Duration()
}

// IdleTimeoutDuration 返回 time.Duration 类型的 IdleTimeout
func (c *RedisConfig) IdleTimeoutDuration() time.Duration {
	return c.IdleTimeout.Duration()
}

// Options 结构体定义了 GORM MySQL 连接器的配置选项（内部使用）
type Options struct {
	Host                  string
	Username              string
	Password              string
	Database              string
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
	LogLevel              logger.LogLevel // 使用 GORM 自带的 LogLevel 类型
	Logger                logger.Interface
	EnableTrace           bool // 是否启用 SQL 追踪插件，用于记录 SQL 执行时间
}

// PostgreSQLOptions 结构体定义了 GORM PostgreSQL 连接器的配置选项（内部使用）
type PostgreSQLOptions struct {
	Host                  string
	Port                  int
	Username              string
	Password              string
	Database              string
	SSLMode               string
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
	LogLevel              logger.LogLevel
	Logger                logger.Interface
	EnableTrace           bool // 是否启用 SQL 追踪插件，用于记录 SQL 执行时间
}

// RedisOptions 结构体定义了 Redis 连接器的配置选项（内部使用）
type RedisOptions struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	EnableTrace  bool // 是否启用命令追踪，用于记录 Redis 命令执行时间
}
