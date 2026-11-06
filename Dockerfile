FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /subgraph ./cmd/subgraph
FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D app
USER app
COPY --from=build /subgraph /usr/local/bin/subgraph
EXPOSE 8080
ENTRYPOINT ["subgraph"]
