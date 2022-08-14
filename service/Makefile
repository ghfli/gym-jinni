
SVCS := user class
DBURL := postgresql://root:gj@127.0.0.1:5432/gj?sslmode=disable&search_path=public

all default: service grpcclt

service: bufgen dbsql gomt
	go build -o cmd/$@

gomt:
	go mod tidy

grpcclt: bufgen
	go build -o cmd/$@ cmd/$@.go

test: all pgup
	export DBURL="${DBURL}"; cmd/service &
	cmd/grpcclt
	cmd/httpclt.sh
	killall -e cmd/service

bufmcc:
	buf mod clear-cache

bufmup:
	for s in ${SVCS} validator ; do buf mod update $$s/pb ; done

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

pgcr:
	sudo docker run -e POSTGRES_PASSWORD=gj -e POSTGRES_USER=root \
		--name postgres -p 5432:5432 -d postgres:alpine
pgdn:
	sudo docker stop postgres
pgup:
	sudo docker start postgres
pgrm: pgdn
	sudo docker rm postgres

dbcr: pgup
	sudo docker exec postgres createdb -U root -O root gj

dbdr: pgup
	sudo docker exec postgres dropdb gj

dbps: pgup
	sudo docker exec -it postgres psql gj

MIGDB = migrate -path mig -verbose -database "${DBURL}" $1 $2
migup: dbsch pgup
	$(call MIGDB,up)
migup1: dbsch pgup
	$(call MIGDB,up,1)
migdn: pgup
	$(call MIGDB,down)
migdn1: pgup
	$(call MIGDB,down,1)
migvn: pgup
	$(call MIGDB,version)
migfr: migvn
	$(call MIGDB,force)  # follow the instructions

setup: pgcr
	while ! sudo docker exec postgres pgrep postgres: -a ; \
		do echo -n . ; sleep 1 ; done
	sleep 30
	$(MAKE) dbcr
	$(MAKE) migup

teardn: migdn dbdr pgrm

bbcr:
	mkdir -p /vagrant/service/bb/data
	sudo docker run -d -t --name budibase -v /vagrant/service/bb/data:/data \
               -p 10000:80 --restart unless-stopped budibase/budibase:latest
bbdn:
	sudo docker stop budibase
bbup:
	sudo docker start budibase

clean:
	rm -f cmd/service cmd/grpcclt dbml-error.log
	rm -rf gen

.PHONY: bufmcc buflint bufmup bufbld bufgen bufpush clean dbsch dbsql gomti \
	pgcr pgdn pgup pgrm dbcr dbdr dbps migup migup1 migdn migdn1 migvn setup \
	teardn
