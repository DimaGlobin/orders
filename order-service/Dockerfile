FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG CMD=cmd
RUN CGO_ENABLED=0 go build -o /bin/app ./${CMD}/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /bin/app /app
ENTRYPOINT ["/app"]
