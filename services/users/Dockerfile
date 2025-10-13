FROM golang:latest

COPY . .

RUN go build -o users .

EXPOSE 8000:8000

EXPOSE 8542:8542

CMD ["./users"]