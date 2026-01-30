# 运行热加载
dev:
	air
username=postgres
password=maotuan
host=192.168.22.227
port=5432
database=urldb
databaseURL="postgres://${username}:${password}@${ host}:${port}/${database}?sslmode=disable"
# 执行数据库升级
migrate-up:
	migrate -path database/migrations -database "postgres://user:pass@localhost:5432/db?sslmode=disable" up
migrate-drop:
	migrate -path database/migrations -database "postgres://user:pass@localhost:5432/db?sslmode=disable" drop -f

# 生成 SQL 代码
generate:
	sqlc generate
build :
				go build -o server.exe .main.go

