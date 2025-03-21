MAKEFLAGS += --no-print-directory

.PHONY: pull
pull:
	git pull

.PHONY: kill
kill:
	sudo docker kill odi-grafana

.PHONY: start
start:
	sudo docker-compose up -d

.PHONY: logs
logs:
	sudo docker logs -f --tail 10 odi-grafana

.PHONY: build
build:
	@$(MAKE) pull
	go mod download
	GOARCH=arm GOARM=7 GOOS=linux go build -o odi-grafana .
	chmod +x odi-grafana


	@if docker ps --format '{{.Names}}' | grep -q '^odi-grafana$$'; then \
		echo "Stopping and removing existing odi-grafana container..."; \
		sudo docker kill odi-grafana; \
		sudo docker rm odi-grafana; \
	else \
		echo "odi-grafana container is not running."; \
	fi

	sudo docker build -t odi-grafana --no-cache .
	sudo docker-compose up -d
	sudo docker logs -f --tail 10 odi-grafana
