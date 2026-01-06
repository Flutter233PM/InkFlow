# InkFlow Backend
FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Defensive: avoid output path being treated as a directory if an `inkflow/` folder exists in build context.
RUN rm -rf /app/inkflow && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/inkflow main.go && \
    test -f /app/inkflow

FROM alpine:3.21
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

RUN mkdir -p /app/config

# Put the binary outside /app so it won't be shadowed even if /app is bind-mounted by an override compose file.
RUN rm -rf /usr/local/bin/inkflow
COPY --from=builder /app/inkflow /usr/local/bin/inkflow

EXPOSE 8080 8081
CMD ["/usr/local/bin/inkflow"]
