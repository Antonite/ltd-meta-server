DB_USER ?= antonite
DB_HOST ?= 127.0.0.1
DB_NAME ?= ltd
DB_PORT ?= 3306
DB_URL ?= 'mysql://$(DB_USER):${DB_PW}@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?query'


.PHONY: migrate
migrate: 
	@ echo "Running migrations..."
	@ migrate -path migrations -database $(DB_URL) up

.PHONY: create-db
create-db:
	@ echo "Creating database..."
	@ mysql -u antonite -e 'CREATE DATABASE $(DB_NAME)'

.PHONY: drop-db
drop-db:
	@ echo "Dropping database..."
	@ mysql -u antonite -e 'DROP DATABASE IF EXISTS $(DB_NAME)'

.PHONY: rebuild-db
rebuild-db: drop-db create-db migrate