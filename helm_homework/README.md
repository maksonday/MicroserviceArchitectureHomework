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
cd manifests && kubectl apply -n miniapp -f .
```

# Как протестить #

TODO