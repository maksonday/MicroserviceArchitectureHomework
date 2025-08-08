# Команды для сборки и пуша в репозиторий #

```bash
docker build --platform linux/amd64 -t miniapp .
docker push maksonday/miniapp
```

# Как протестить приложение #

```bash
docker run -p 8000:8000 maksonday/miniapp &
curl -v localhost:8000/health
```

## Пример ответа ##
```
* Host localhost:8000 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:8000...
* Connected to localhost (::1) port 8000
> GET /health HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 200 OK
< Date: Tue, 08 Jul 2025 18:08:38 GMT
< Content-Length: 0
< 
* Connection #0 to host localhost left intact
{"status":"OK"}
```
