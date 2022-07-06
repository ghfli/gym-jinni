

MODS := user

all default: service grpcclt

service: bufgen dbsch
	go build -o cmd/$@

grpcclt: bufgen
	go build -o cmd/$@ cmd/$@.go

test: all
	cmd/service &
	cmd/grpcclt
	cmd/httpclt.sh
	killall service

bufmcc:
	buf mod clear-cache

bufmup:
	for m in ${MODS} ; do buf mod update $$m ; done

buflint: bufmup
	for m in ${MODS} ; do buf lint $$m ; done

bufbld: bufmup
	for m in ${MODS} ; do buf build $$m ; done

bufgen: bufbld
	for m in ${MODS} ; do buf generate $$m ; done

bufpush: bufbld
	for m in ${MODS} ; do buf push $$m ; done

dbsch:
	mkdir -p gen/sql
	for m in ${MODS} ; do \
		dbml2sql --postgres -o gen/sql/$$m.sql $$m/$$m.dbml ; \
	done

gomt:
	go mod tidy

clean:
	rm -f cmd/service cmd/grpcclt

.PHONY: bufmcc buflint bufmup bufbld bufgen bufpush clean dbsch gomti
