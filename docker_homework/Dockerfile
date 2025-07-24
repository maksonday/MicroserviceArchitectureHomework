FROM golang:latest

COPY . .

RUN go build -o miniapp .

EXPOSE 8000:8000

CMD ["./miniapp"]