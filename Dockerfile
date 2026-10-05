FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/auth-v2 ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/auth-v2 /auth-v2
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/auth-v2"]
