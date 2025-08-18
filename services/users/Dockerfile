FROM golang:latest

COPY . .

RUN go build -o users .

EXPOSE 8000:8000

CMD ["./users"]
#CMD ["tail", "-f", "/dev/null"]