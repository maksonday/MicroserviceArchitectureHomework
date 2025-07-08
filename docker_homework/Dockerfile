FROM golang:latest

COPY . .

RUN go build main.go

EXPOSE 8000:8000

CMD ["./main",  "&"]