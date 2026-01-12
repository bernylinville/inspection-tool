module inspection-tool/apps/cmdb-server

go 1.25.5

require (
	inspection-tool/pkg v0.0.0
	github.com/gin-gonic/gin v1.11.0
	gorm.io/gorm v1.31.1
	gorm.io/driver/postgres v1.6.0
)

replace inspection-tool/pkg => ../../pkg
