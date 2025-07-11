# Команды для разворачивания #

1. Прописываем в /etc/hosts для `minikube ip` хост `arch.homework`

2. Создаем namespace `miniapp`, в котором устанавливаем контроллер ingress-nginx

```bash
kubectl create namespace miniapp && helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx/ && helm repo update && helm install nginx ingress-nginx/ingress-nginx --namespace miniapp -f nginx-ingress.yaml
```

3. Применяем манифесты

```bash
kubectl apply -f .
```

# Как протестить #

```bash
curl http://arch.homework/health
```

## Пример ответа ##
```json
{"status":"OK"}
```