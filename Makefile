COMPOSE_FILE=./docker-compose.yaml
ENV_FILE=./.env

DC=docker-compose -f ${COMPOSE_FILE} --env-file ${ENV_FILE}
APP=enlabs-test_app
SERVICES=postgres enlabs-test_app
TEST_REQUESTS=enlabs-run_tests
ALL=${SERVICES} migrate
UNIT=enlabs-unit
ifneq (,$(wildcard ${ENV_FILE}))
    include .env
    export
endif

.PHONY: build
build:
	${DC} build

.PHONY: up
up:
	@echo "Starting Docker images..."
	${DC} up -d --remove-orphans ${SERVICES}
	@echo "Docker images started!"

.PHONY: up_build
up_build:
	@echo "Stopping docker images (if running...)"
	${DC} down
	@echo "Building (when required) and starting docker images..."
	${DC} up --build -d --remove-orphans ${SERVICES}
	@echo "Docker images built and started!"

.PHONY: up_build_all
up_build_all:
	@echo "Stopping docker images (if running...)"
	$(MAKE) down
	@echo "Building (when required) and starting docker images..."
	${DC} up --build -d --remove-orphans ${ALL}
	@echo "Docker images built and started!"

.PHONY: down
down:
	@echo "Stopping docker images..."
	${DC} down --remove-orphans --volumes
	@echo "Docker images stopped!"

.PHONY: restart
restart:
	@echo "Restarting docker images..."
	${DC} restart ${SERVICES}
	@echo "Docker images restarted!"

.PHONY: logs
logs:
	@echo "Showing logs..."
	${DC} logs -f

.PHONY: ps
ps:
	@echo "Showing running containers..."
	${DC} ps

.PHONY: install
install:
	@echo "Running migrations..."
	${DC} up -d --remove-orphans ${ALL}
	@echo "Migrations applied!"

.PHONY: test_requests
test_requests:
	@echo "Running tests..."
	${DC} up --build --force-recreate -d --remove-orphans ${TEST_REQUESTS}
	${DC} logs -f ${TEST_REQUESTS}
	@echo "Tests passed!"

.PHONY: test_unit
test_unit:
	@echo "Building and running unit tests in Docker container..."
	${DC} up --build -d ${UNIT}
	docker exec -it ${PROJECT_NAME}_${UNIT} sh -c "CGO_ENABLED=0 go test -v ./internal/infrastructure/database/repositories/..."
	@echo "Unit tests completed!"
