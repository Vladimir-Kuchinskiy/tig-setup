FROM golang:1.21-alpine

RUN apk add curl

WORKDIR /app

COPY . .

RUN go build -o app

EXPOSE 8080

CMD ["./app"]