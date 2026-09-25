FROM golang:1.27.1-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/proxy ./cmd/server

FROM alpine:3.24

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/proxy /app/proxy
EXPOSE 8000
CMD ["/app/proxy"]
