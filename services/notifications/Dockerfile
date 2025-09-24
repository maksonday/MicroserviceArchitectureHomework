FROM golang:latest

COPY . .

RUN go build -o stock .

EXPOSE 8000:8000

CMD ["./stock"]