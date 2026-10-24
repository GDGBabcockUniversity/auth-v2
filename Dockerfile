FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY docs ./docs
RUN CGO_ENABLED=0 go build -trimpath -o /auth-api ./cmd/auth

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 65532 nonroot \
    && adduser -S -D -H -u 65532 -G nonroot nonroot
COPY --from=build /auth-api /auth-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/auth-api"]
