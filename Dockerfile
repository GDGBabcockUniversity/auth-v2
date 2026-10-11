FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /auth-api ./cmd/auth

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /auth-api /auth-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/auth-api"]
