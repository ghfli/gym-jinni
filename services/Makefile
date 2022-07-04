

BUF_MODS := usermgt

all default: services grpcclt

services: bufgen
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

buflint:
	for m in ${BUF_MODS} ; do buf lint $$m ; done

bufmup: buflint
	for m in ${BUF_MODS} ; do buf mod update $$m ; done

bufbld: bufmup
	for m in ${BUF_MODS} ; do buf build $$m ; done

bufgen: bufbld
	for m in ${BUF_MODS} ; do buf generate $$m ; done

bufpush: bufbld
	for m in ${BUF_MODS} ; do buf push $$m ; done

gomti:
	go mod tidy

clean:
	rm -f services grpcclt

.PHONY: bufmcc buflint bufmup bufbld bufgen bufpush clean gomti
