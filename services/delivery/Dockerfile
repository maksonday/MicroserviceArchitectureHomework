FROM golang:latest

COPY . .

RUN go build -o delivery .

EXPOSE 8000:8000

CMD ["./delivery"]