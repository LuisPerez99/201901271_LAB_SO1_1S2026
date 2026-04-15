# Guía de Deployment M.U.M.N.K8s

## Checklist de Pre-deployment

- [ ] Cluster GKE disponible con N1 instances
- [ ] KubeVirt instalado en el cluster
- [ ] Zot registry configurado con HTTPS
- [ ] Credenciales de Zot disponibles
- [ ] git clone del repositorio

## Fase 1: Build y Push de Imágenes

### Rust API
```bash
cd proyecto3/api
docker build -t rust-api:v1 .
# Tag y push a Zot
docker tag rust-api:v1 zot:5000/api-rust:v1
docker push zot:5000/api-rust:v1
```

### Go API Client
```bash
cd proyecto3/go-api-client
docker build -t go-api-client:v1 .
docker tag go-api-client:v1 zot:5000/go-api-client:v1
docker push zot:5000/go-api-client:v1
```

### Go gRPC Server
```bash
cd proyecto3/go-grpc-server
docker build -t go-grpc-server:v1 .
docker tag go-grpc-server:v1 zot:5000/go-grpc-server:v1
docker push zot:5000/go-grpc-server:v1
```

### Go Consumer
```bash
cd proyecto3/go-consumer
docker build -t go-consumer:v1 .
docker tag go-consumer:v1 zot:5000/go-consumer:v1
docker push zot:5000/go-consumer:v1
```

## Fase 2: Deploy a Kubernetes

### Crear namespace
```bash
kubectl apply -f k8s/01-namespace.yaml
```

### Deploy componentes base
```bash
kubectl apply -f k8s/02-rust-api.yaml
kubectl apply -f k8s/03-rabbitmq.yaml
kubectl apply -f k8s/04-go-api-client.yaml
kubectl apply -f k8s/05-go-grpc-server.yaml
kubectl apply -f k8s/06-go-consumer.yaml
```

### Verificar estado
```bash
kubectl get pods -n proyecto3
kubectl get svc -n proyecto3
```

### Deploy Gateway API
```bash
kubectl apply -f k8s/08-gateway-api.yaml
```

### Deploy HPA 
```bash
kubectl apply -f k8s/07-hpa.yaml
```

## Fase 3: Testing

### Obtener IP del Gateway
```bash
kubectl get gateway -n proyecto3
kubectl get svc -A | grep gateway
```

### Port-forward a Rust API
```bash
kubectl port-forward svc/rust-api -n proyecto3 3000:3000
```

### Enviar reporte de prueba
```bash
curl -X POST http://localhost:3000/report \
  -H "Content-Type: application/json" \
  -d '{
    "country": "USA",
    "warplanes_in_air": 25,
    "warships_in_water": 12,
    "timestamp": "2026-04-14T10:30:00Z"
  }'
```

## Troubleshooting

### Ver logs de un pod
```bash
kubectl logs -n proyecto3 <pod-name> -f
```

### Shell en un pod
```bash
kubectl exec -it -n proyecto3 <pod-name> -- /bin/sh
```

### Checking RabbitMQ
```bash
kubectl port-forward -n proyecto3 svc/rabbitmq 15672:15672
# Acceder a http://localhost:15672 (guest/guest)
```

### Restart deployment
```bash
kubectl rollout restart deployment/<name> -n proyecto3
```

## Próximos pasos

1. [ ] Completar implementación de gRPC en Go services
2. [ ] Setup KubeVirt VMs (Valkey + Grafana)
3. [ ] Conectar Grafana a Valkey
4. [ ] Load testing con Locust
5. [ ] Análisis y documentación
