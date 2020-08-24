DC=docker-compose
DE=docker-compose exec -T app
IMAGE=dkr.hanaboso.net/hanaboso/go-elasticsearch

.env:
	sed -e 's/{DEV_UID}/$(shell id -u)/g' \
		-e 's/{DEV_GID}/$(shell id -g)/g' \
		-e 's|{DOCKER_SOCKET_PATH}|$(shell test -S /var/run/docker-$${USER}.sock && echo /var/run/docker-$${USER}.sock || echo /var/run/docker.sock)|g' \
		.env.dist >> .env; \

docker-up-force: .env
	$(DC) pull
	$(DC) up -d --force-recreate --remove-orphans

docker-down-clean: .env
	$(DC) down -v

docker-compose.ci.yml:
	# Comment out any port forwarding
	sed -r 's/^(\s+ports:)$$/#\1/g; s/^(\s+- \$$\{DEV_IP\}.*)$$/#\1/g; s/^(\s+- \$$\{GOPATH\}.*)$$/#\1/g' docker-compose.yml > docker-compose.ci.yml

go-update:
	$(DE) su-exec root go get -u all
	$(DE) su-exec root go mod tidy
	$(DE) su-exec root chown dev:dev go.mod go.sum

init-dev: docker-up-force wait-for-it-db

wait-for-it-db:
	$(DE) /bin/sh -c 'while [ $$(curl -s -o /dev/null -w "%{http_code}" http://elasticsearch01:9200) == 000 ]; do sleep 1; done'
	$(DE) /bin/sh -c 'while [ $$(curl -s -o /dev/null -w "%{http_code}" http://elasticsearch02:9200) == 000 ]; do sleep 1; done'
	$(DE) /bin/sh -c 'while [ $$(curl -s -o /dev/null -w "%{http_code}" http://elasticsearch03:9200) == 000 ]; do sleep 1; done'

lint:
	$(DE) go fmt ./...
	$(DE) goimports -local elasticsearch -w --format-only .
	excludes='';\
	for file in $$(ls -R $$(find . -type f ) | grep test.go); do\
		excludes="$${excludes} -exclude $$(echo $${file} | cut -c 3-)";\
	done;\
	$(DE) revive -config config.toml $${excludes} -formatter friendly ./...

fast-test: lint
	$(DE) mkdir var || true
	$(DE) go test -p 1 -count 1 --parallel 1 -cover -coverprofile var/coverage.out ./...
	$(DE) go tool cover -html=var/coverage.out -o var/coverage.html

coverage:
	$(DE) /coverage.sh -c 85
	$(DE) go tool cover -html=var/coverage.out -o var/coverage.html

test: init-dev fast-test coverage docker-down-clean
