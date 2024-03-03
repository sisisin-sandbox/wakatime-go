FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main -tags timetzdata ./src/cmd/downloader


FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

ENTRYPOINT ["/app/main"]
CMD ["./main"]
