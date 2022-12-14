postgres:
	docker run --name postgres14.3 --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14.3-alpine

createdb:
	docker exec -it postgres14.3 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres14.3 dropdb simple_bank

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up

migrateupaws:
	migrate -path db/migration -database "postgresql://root:UompK6LennLrdNPejFDl@simple-bank.cxponydhewvn.us-east-1.rds.amazonaws.com:5432/simple_bank" -verbose up

migrateup1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down

migratedown1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down 1

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

mock: 
	mockgen -package mockdb -destination db/mock/store.go github.com/techschool/simplebank/db/sqlc Store

dbdocs:
	dbdocs build doc/db.dbml

dbschema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml

proto:
	rm -f pb/*.go
	rm -f doc/swagger/*.swagger.json
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative proto/*.proto
	protoc --proto_path=proto --go-grpc_out=pb --go-grpc_opt=paths=source_relative proto/*.proto
	protoc --proto_path=proto --grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative proto/*.proto
	protoc --proto_path=proto --openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank proto/*.proto
	statik -src=./doc/swagger -dest=./doc 

evans:
	evans --host localhost --port 9090 -r repl

.PHONY: postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 sqlc test server mock migrateupaws dbdocs dbschema proto evans