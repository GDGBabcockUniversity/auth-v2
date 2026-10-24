.PHONY: swagger
swagger:
	swag init -g cmd/auth/main.go -o docs --parseDependency --parseInternal