FROM golang:1.21-alpine as builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . ./
RUN go build -o /usr/bin/dgate-server ./cmd/dgate-server
RUN go build -o /usr/bin/dgate-cli ./cmd/dgate-cli

FROM alpine:latest as runner
WORKDIR /app
COPY --from=builder /usr/bin/dgate-server ./
COPY --from=builder /usr/bin/dgate-cli ./
COPY --from=builder /app/config.dgate.yaml ./
EXPOSE 80 443 9080 9443
CMD [ "./dgate-server" ]