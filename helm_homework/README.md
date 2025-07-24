# Репозиторий с чартом

https://github.com/maksonday/miniapp

# Команды для разворачивания #

1. Открываем порт 8443 для всех нод minikube(нужно для запуска ingress-nginx admission controller)

```bash
sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
```

2. Прописываем в /etc/hosts для `minikube ip` хост `arch.homework`

3. Создаем Secret, в котором прописан пароль для пользователя postgres в base64

```bash
cd manifests && kubectl apply -n miniapp -f secret.yaml
```

4. Устанавливаем чарт

* Из локальных файлов
```bash
helm -n miniapp install ./users-crud --generate-name --set postgresql.auth.existingSecret=postgres-secret
```

* Из репозитория
```bash
helm repo add maksonday https://maksonday.github.io/miniapp/
helm install users-crud --set postgresql.auth.existingSecret=postgres-secret maksonday/users-crud -n minapp
```

5. Запускаем тесты

```bash
newman run users-crud.postman_collection.json

newman

users-crud

→ healthcheck
  GET arch.homework/health [200 OK, 156B, 40ms]
  ✓  Response status code is 200
  ✓  Response body is equal to expected JSON

→ user not created yet
  GET arch.homework/user/1 [400 Bad Request, 167B, 8ms]
  ✓  Response status code is 400
  ✓  Response body is equal to expected JSON

→ create user
  POST arch.homework/user/ [201 Created, 144B, 7ms]
  ✓  Response status code is 201
  ✓  Response body contains key 'id' and sets environment variable

→ get created user
  GET arch.homework/user/1 [200 OK, 229B, 4ms]
  ✓  Response status code is 200
  ✓  Response body is equal to expected JSON

→ update user
  POST arch.homework/user/1 [200 OK, 99B, 6ms]
  ✓  Response status code is 200

→ New Request
  DELETE arch.homework/user/1 [204 No Content, 88B, 4ms]
  ✓  Response status code is 204

┌─────────────────────────┬──────────────────┬──────────────────┐
│                         │         executed │           failed │
├─────────────────────────┼──────────────────┼──────────────────┤
│              iterations │                1 │                0 │
├─────────────────────────┼──────────────────┼──────────────────┤
│                requests │                6 │                0 │
├─────────────────────────┼──────────────────┼──────────────────┤
│            test-scripts │               12 │                0 │
├─────────────────────────┼──────────────────┼──────────────────┤
│      prerequest-scripts │                6 │                0 │
├─────────────────────────┼──────────────────┼──────────────────┤
│              assertions │               10 │                0 │
├─────────────────────────┴──────────────────┴──────────────────┤
│ total run duration: 266ms                                     │
├───────────────────────────────────────────────────────────────┤
│ total data received: 146B (approx)                            │
├───────────────────────────────────────────────────────────────┤
│ average response time: 11ms [min: 4ms, max: 40ms, s.d.: 12ms] │
└───────────────────────────────────────────────────────────────┘
```