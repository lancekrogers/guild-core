#!/bin/bash
# Scenario 5: Deployment with DeploymentMarshal
# Duration: 4-5 minutes
# Purpose: Show production deployment configuration

echo "🏰 Guild Framework Demo - Scenario 5: Production Deployment"
echo "========================================================="
echo ""
echo "🚀 Working with DeploymentMarshal on Kubernetes deployment..."
echo ""

echo "$ guild chat --campaign e-commerce"
echo ""
echo "You: @deployment-marshal Create production Kubernetes deployment with monitoring and auto-scaling"
echo ""
sleep 1

cat << 'EOF'
🤔 deployment-marshal is thinking...
   Planning production deployment strategy...

DeploymentMarshal: I'll create a production-ready Kubernetes deployment with monitoring, auto-scaling, and zero-downtime updates. Let me build a comprehensive solution.

⚙️ deployment-marshal is working...
   Creating deployment manifests...
   [████░░░░░░░░░░░░] 25%

## Production Kubernetes Deployment

### 1. Namespace and ConfigMap

```yaml
# k8s/namespaces/ecommerce.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ecommerce-prod
  labels:
    name: ecommerce-prod
    environment: production
---
# k8s/configmaps/common-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: common-config
  namespace: ecommerce-prod
data:
  LOG_LEVEL: "info"
  TRACE_ENABLED: "true"
  METRICS_PORT: "9090"
```

⚙️ deployment-marshal is working...
   Configuring auth service deployment...
   [████████░░░░░░░░] 50%

### 2. Auth Service Deployment

```yaml
# k8s/deployments/auth-service.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: auth-service
  namespace: ecommerce-prod
  labels:
    app: auth-service
    version: v1
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: auth-service
  template:
    metadata:
      labels:
        app: auth-service
        version: v1
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: auth-service
        image: ecommerce/auth-service:1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-credentials
              key: auth-db-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: auth-secrets
              key: jwt-secret
        envFrom:
        - configMapRef:
            name: common-config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 15"]
---
apiVersion: v1
kind: Service
metadata:
  name: auth-service
  namespace: ecommerce-prod
spec:
  selector:
    app: auth-service
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
```

⚙️ deployment-marshal is working...
   Setting up auto-scaling...
   [████████████░░░░] 75%

### 3. Horizontal Pod Autoscaler

```yaml
# k8s/hpa/auth-service-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: auth-service-hpa
  namespace: ecommerce-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: auth-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "1000"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
      - type: Pods
        value: 2
        periodSeconds: 60
```

### 4. Ingress Configuration

```yaml
# k8s/ingress/api-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
  namespace: ecommerce-prod
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/ssl-protocols: "TLSv1.2 TLSv1.3"
spec:
  tls:
  - hosts:
    - api.ecommerce.com
    secretName: api-tls-cert
  rules:
  - host: api.ecommerce.com
    http:
      paths:
      - path: /api/v1/auth
        pathType: Prefix
        backend:
          service:
            name: auth-service
            port:
              number: 80
```

### 5. Monitoring Stack

```yaml
# k8s/monitoring/prometheus-servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: ecommerce-services
  namespace: ecommerce-prod
spec:
  selector:
    matchLabels:
      monitoring: enabled
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
---
# k8s/monitoring/grafana-dashboard.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards
  namespace: monitoring
data:
  ecommerce-dashboard.json: |
    {
      "dashboard": {
        "title": "E-Commerce Platform Metrics",
        "panels": [
          {
            "title": "Request Rate",
            "targets": [
              {
                "expr": "rate(http_requests_total[5m])"
              }
            ]
          },
          {
            "title": "Error Rate",
            "targets": [
              {
                "expr": "rate(http_requests_total{status=~\"5..\"}[5m])"
              }
            ]
          },
          {
            "title": "Response Time P95",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, http_request_duration_seconds_bucket)"
              }
            ]
          }
        ]
      }
    }
```

### 6. CI/CD Pipeline

```yaml
# .github/workflows/deploy.yaml
name: Deploy to Production
on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Build and Push Docker Image
      run: |
        docker build -t ecommerce/auth-service:${GITHUB_REF#refs/tags/} .
        docker push ecommerce/auth-service:${GITHUB_REF#refs/tags/}
    
    - name: Deploy to Kubernetes
      run: |
        kubectl set image deployment/auth-service \
          auth-service=ecommerce/auth-service:${GITHUB_REF#refs/tags/} \
          -n ecommerce-prod
        
        kubectl rollout status deployment/auth-service -n ecommerce-prod
    
    - name: Run Smoke Tests
      run: |
        ./scripts/smoke-tests.sh https://api.ecommerce.com
```

✅ deployment-marshal completed deployment configuration

DeploymentMarshal: Production deployment ready with:

1. **High Availability**: 3+ replicas with pod disruption budgets
2. **Auto-scaling**: CPU, memory, and custom metrics
3. **Zero-downtime**: Rolling updates with readiness checks
4. **Monitoring**: Prometheus metrics and Grafana dashboards
5. **Security**: TLS termination, network policies
6. **CI/CD**: Automated deployment pipeline

Deploy with: `kubectl apply -k k8s/overlays/production`
EOF

echo ""
echo "📊 Deployment Summary:"
echo "- Created Kubernetes manifests for all services"
echo "- Configured auto-scaling with custom metrics"
echo "- Set up Prometheus monitoring"
echo "- Implemented zero-downtime deployment strategy"
echo ""
echo "🎯 DeploymentMarshal's expertise demonstrated:"
echo "- Container orchestration best practices"
echo "- Production-grade configurations"
echo "- Monitoring and observability setup"
echo "- CI/CD pipeline integration"