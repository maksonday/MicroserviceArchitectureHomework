FROM golang:latest

COPY . .

RUN go build -o order .

EXPOSE 8000:8000

CMD ["./order"]