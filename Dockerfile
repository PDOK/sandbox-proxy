FROM docker.io/library/golang:1.15.7 as builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

RUN go test ./...

# Build linux binary
RUN CGO_ENABLED=0 GOOS=linux \
  go build  -ldflags '-w -s' -a -installsuffix cgo -o /builds/linux/sandbox-proxy sandbox-proxy

FROM scratch as sandbox-proxy

WORKDIR /tmp

COPY --from=builder /builds/linux/sandbox-proxy /bin/

EXPOSE 5000
EXPOSE 5001
EXPOSE 5002
EXPOSE 5003
EXPOSE 5004
EXPOSE 5005
EXPOSE 5006

ENV BIND_ADDRESS=0.0.0.0

ENTRYPOINT [ "sandbox-proxy" ]
