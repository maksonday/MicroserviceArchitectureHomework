FROM golang:latest

COPY . .

RUN go build -o auth-service .

EXPOSE 8000:8000

CMD ["./auth-service"]
#CMD ["tail", "-f", "/dev/null"]