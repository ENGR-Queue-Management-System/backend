FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . /app

RUN go mod tidy
RUN go build -o main .


FROM alpine:latest  

WORKDIR /app

COPY --from=builder /app/main /app/
COPY .env /app

EXPOSE 8000

CMD ["./main"]
