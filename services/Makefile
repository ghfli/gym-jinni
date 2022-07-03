

BUF_MODS := usermgt

all default: services grpcclt

services: buf_gen
	go build

grpcclt: buf_gen
	go build test/grpcclt.go

test: all
	./services &
	./grpcclt
	test/httpclt.sh
	killall services

buf_mcc:
	buf mod clear-cache

buf_mup:
	for m in ${BUF_MODS} ; do buf mod update $$m ; done

buf_bld: buf_mup
	for m in ${BUF_MODS} ; do buf build $$m ; done

buf_gen: buf_bld
	for m in ${BUF_MODS} ; do buf generate $$m ; done

buf_push:
	for m in ${BUF_MODS} ; do buf push $$m ; done

go_mti:
	go mod tidy

clean:
	rm -f services grpcclt

.PHONY: buf_mcc buf_mup buf_bld buf_gen buf_push clean go_mti
