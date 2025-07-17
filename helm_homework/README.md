# Команды для разворачивания #

1. Прописываем в /etc/hosts для `minikube ip` хост `arch.homework`

2. Создаем namespace `miniapp`, в котором устанавливаем контроллер ingress-nginx

```bash
kubectl create namespace miniapp && helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx/ && helm repo update && helm install nginx ingress-nginx/ingress-nginx --namespace miniapp -f nginx-ingress.yaml
```

3. Создаем Secret

```bash
cd manifests && kubectl apply -n miniapp -f secret.yaml
```

4. Разворачиваем Postgres в том же namespace

```bash
cd .. && helm install my-release oci://registry-1.docker.io/bitnamicharts/postgresql -n miniapp --values values.yaml
```

5. Применяем манифесты

```bash
kubectl apply -n miniapp -f manifests/
```

Если появляется ошибка:
```bash
Error from server (InternalError): error when creating "manifests/ingress.yaml": Internal error occurred: failed calling webhook "validate.nginx.ingress.kubernetes.io": failed to call webhook: Post "https://ingress-nginx-controller-admission.ingress-nginx.svc:443/networking/v1/ingresses?timeout=10s": service "ingress-nginx-controller-admission" not found
```

Удаляем вебхук

```bash
kubectl delete -A ValidatingWebhookConfiguration ingress-nginx-admission
```

Заново применяем манифесты

# Как протестить #

```bash
newman run users-crud.postman_collection.json
```