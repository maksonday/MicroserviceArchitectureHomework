FROM golang:latest

COPY . .

RUN go build -o notifications .

EXPOSE 8000:8000

CMD ["./notifications"]