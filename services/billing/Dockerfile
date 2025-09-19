FROM golang:latest

COPY . .

RUN go build -o billing .

EXPOSE 8000:8000

CMD ["tail", "-f", "/dev/null"]