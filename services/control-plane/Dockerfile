FROM golang:1.23-alpine AS build

WORKDIR /src

COPY services/control-plane/go.mod services/control-plane/go.sum* ./
RUN go mod download

COPY services/control-plane/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/control-plane ./cmd/api

FROM alpine:3.20

RUN adduser -D -u 10001 appuser

COPY --from=build /out/control-plane /usr/local/bin/control-plane

USER appuser
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/control-plane"]

