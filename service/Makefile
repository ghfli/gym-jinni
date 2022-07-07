
SVCS := user class

all default: service grpcclt

service: bufgen dbsql gomt
	go build -o cmd/$@

grpcclt: bufgen
	go build -o cmd/$@ cmd/$@.go

test: all
	cmd/service &
	cmd/grpcclt
	cmd/httpclt.sh
	killall -e cmd/service

bufmcc:
	buf mod clear-cache

bufmup:
	for s in ${SVCS} ; do buf mod update $$s/pb ; done

buflint: bufmup
	buf lint

bufbld: bufmup
	buf build

bufgen: bufbld
	buf generate

bufpush: bufbld
	buf push

dbsch:
	mkdir -p gen/sql
	for s in ${SVCS} ; do \
		dbml2sql --postgres -o gen/sql/$$s.dbml.sql $$s/db/$$s.dbml ; \
	done

dbsql: dbsch
	for s in ${SVCS} ; do (cd $$s/db && sqlc generate) ; done

gomt:
	go mod tidy

clean:
	rm -f cmd/service cmd/grpcclt dbml-error.log
	rm -rf gen

.PHONY: bufmcc buflint bufmup bufbld bufgen bufpush clean dbsch dbsql gomti
