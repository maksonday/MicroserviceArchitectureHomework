FROM golang:latest

COPY . .

RUN go build -o auth-service .

EXPOSE 8000:8000

EXPOSE 8542:8542

CMD ["./auth-service"]