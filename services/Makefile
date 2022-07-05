

MODS := usermgt

all default: services grpcclt

services: bufgen dbsch
	go build

grpcclt: bufgen
	go build test/grpcclt.go

test: all
	./services &
	./grpcclt
	test/httpclt.sh
	killall services

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
	rm -f services grpcclt

.PHONY: bufmcc buflint bufmup bufbld bufgen bufpush clean dbsch gomti
