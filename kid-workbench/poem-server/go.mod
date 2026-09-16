module github.com/conchi/poem-server

go 1.26.1

require (
	github.com/conchi/study-learning v0.0.0
	github.com/gin-gonic/gin v1.12.0
	github.com/glebarez/sqlite v1.11.0
	github.com/stretchr/testify v1.12.1
	gorm.io/driver/postgres v1.6.2
	gorm.io/gorm v1.31.2
)

replace github.com/conchi/study-learning => ../shared-go
