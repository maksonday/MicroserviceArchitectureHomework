FROM golang:latest

COPY . .

RUN go build main.go

RUN ./main &

EXPOSE 8000:8000

CMD ["./main",  "&"]