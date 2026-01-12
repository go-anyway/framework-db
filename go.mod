module github.com/go-anyway/framework-db

go 1.25.4

require (
	github.com/go-anyway/framework-config v1.0.0
	github.com/go-anyway/framework-log v1.0.0
	github.com/go-anyway/framework-metrics v1.0.0
	github.com/go-anyway/framework-trace v1.0.0
	github.com/redis/go-redis/v9 v9.17.2
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
	gorm.io/plugin/dbresolver v1.6.2
)

replace (
	github.com/go-anyway/framework-config => ../config
	github.com/go-anyway/framework-log => ../log
	github.com/go-anyway/framework-metrics => ../../optional/metrics
	github.com/go-anyway/framework-trace => ../../optional/trace
)
