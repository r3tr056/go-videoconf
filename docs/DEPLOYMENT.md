# VideoConf - Production Deployment Guide

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   Kubernetes    │    │   Monitoring    │
│    (Nginx)      │    │   Cluster       │    │  (Prometheus)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐              │
         │              │  Ingress Ctrl   │              │
         │              └─────────────────┘              │
         │                       │                       │
    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
    │   Users Service │    │Signalling Server│    │   Media Relay   │
    │   (Port 8081)   │    │   (Port 8080)   │    │   (TURN/STUN)   │
    └─────────────────┘    └─────────────────┘    └─────────────────┘
             │                       │                       │
             │              ┌─────────────────┐              │
             │              │    MongoDB      │              │
             │              │  (Replica Set)  │              │
             │              └─────────────────┘              │
             │                                               │
    ┌─────────────────┐                            ┌─────────────────┐
    │     Redis       │                            │     CDN         │
    │ (Session Store) │                            │  (Static Files) │
    └─────────────────┘                            └─────────────────┘
```

## 🚀 Deployment Options

### Option 1: Docker Compose (Development/Small Scale)

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.prod.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl
    depends_on:
      - signalling-server
      - users-service
      - client
    restart: unless-stopped

  signalling-server:
    build: ./server/signalling-server
    environment:
      - MONGODB_URI=mongodb://mongodb:27017/videoconf
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - TURN_SECRET=${TURN_SECRET}
    depends_on:
      - mongodb
      - redis
    restart: unless-stopped
    deploy:
      replicas: 3

  users-service:
    build: ./server/users-service
    environment:
      - MONGODB_URI=mongodb://mongodb:27017/videoconf
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - mongodb
    restart: unless-stopped
    deploy:
      replicas: 2

  client:
    build: ./client
    environment:
      - REACT_APP_SIGNALLING_SERVER=wss://your-domain.com
      - REACT_APP_USERS_API=https://your-domain.com/api/users
    restart: unless-stopped

  mongodb:
    image: mongo:5.0
    environment:
      - MONGO_INITDB_ROOT_USERNAME=${MONGO_ROOT_USERNAME}
      - MONGO_INITDB_ROOT_PASSWORD=${MONGO_ROOT_PASSWORD}
    volumes:
      - mongodb_data:/data/db
      - ./scripts/mongo-init.js:/docker-entrypoint-initdb.d/init.js
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    restart: unless-stopped

  coturn:
    image: coturn/coturn:latest
    network_mode: host
    environment:
      - TURN_SECRET=${TURN_SECRET}
    volumes:
      - ./coturn.conf:/etc/coturn/turnserver.conf
    restart: unless-stopped

volumes:
  mongodb_data:
  redis_data:
```

### Option 2: Kubernetes (Production Scale)

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: videoconf
---
# k8s/mongodb.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mongodb
  namespace: videoconf
spec:
  serviceName: mongodb
  replicas: 3
  selector:
    matchLabels:
      app: mongodb
  template:
    metadata:
      labels:
        app: mongodb
    spec:
      containers:
      - name: mongodb
        image: mongo:5.0
        ports:
        - containerPort: 27017
        env:
        - name: MONGO_INITDB_ROOT_USERNAME
          valueFrom:
            secretKeyRef:
              name: mongodb-secret
              key: username
        - name: MONGO_INITDB_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mongodb-secret
              key: password
        volumeMounts:
        - name: mongodb-storage
          mountPath: /data/db
  volumeClaimTemplates:
  - metadata:
      name: mongodb-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi
---
# k8s/signalling-server.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: signalling-server
  namespace: videoconf
spec:
  replicas: 5
  selector:
    matchLabels:
      app: signalling-server
  template:
    metadata:
      labels:
        app: signalling-server
    spec:
      containers:
      - name: signalling-server
        image: ghcr.io/r3tr056/go-videoconf/signalling-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: MONGODB_URI
          value: mongodb://mongodb:27017/videoconf
        - name: REDIS_URL
          value: redis://redis:6379
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: jwt-secret
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: signalling-server
  namespace: videoconf
spec:
  selector:
    app: signalling-server
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
---
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: videoconf-ingress
  namespace: videoconf
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    nginx.ingress.kubernetes.io/websocket-services: "signalling-server"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - your-domain.com
    secretName: videoconf-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /api/signalling
        pathType: Prefix
        backend:
          service:
            name: signalling-server
            port:
              number: 8080
      - path: /api/users
        pathType: Prefix
        backend:
          service:
            name: users-service
            port:
              number: 8081
      - path: /
        pathType: Prefix
        backend:
          service:
            name: client
            port:
              number: 80
```

## 🔧 Configuration

### Environment Variables

```bash
# Production environment file (.env.prod)

# Database
MONGODB_URI=mongodb://user:password@mongodb:27017/videoconf?authSource=admin
REDIS_URL=redis://redis:6379

# Security
JWT_SECRET=your-super-secure-jwt-secret-256-bits
TURN_SECRET=your-turn-server-secret

# Services
SIGNALLING_SERVER_PORT=8080
USERS_SERVICE_PORT=8081

# External Services
TURN_SERVER_URL=turn:your-turn-server.com:3478
STUN_SERVER_URL=stun:stun.l.google.com:19302

# Monitoring
PROMETHEUS_ENDPOINT=http://prometheus:9090
GRAFANA_ENDPOINT=http://grafana:3000

# CDN
CDN_URL=https://cdn.your-domain.com
STATIC_FILES_URL=https://static.your-domain.com

# SSL
SSL_CERT_PATH=/etc/ssl/certs/your-domain.crt
SSL_KEY_PATH=/etc/ssl/private/your-domain.key
```

### NGINX Configuration

```nginx
# nginx.prod.conf
events {
    worker_connections 1024;
}

http {
    upstream signalling_backend {
        server signalling-server:8080;
        server signalling-server:8080;
        server signalling-server:8080;
    }

    upstream users_backend {
        server users-service:8081;
        server users-service:8081;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=websocket:10m rate=5r/s;

    server {
        listen 80;
        server_name your-domain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/ssl/certs/your-domain.crt;
        ssl_certificate_key /etc/ssl/private/your-domain.key;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;

        # WebSocket upgrade
        location /ws {
            limit_req zone=websocket burst=20 nodelay;
            proxy_pass http://signalling_backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 86400;
        }

        # API routes
        location /api/signalling {
            limit_req zone=api burst=50 nodelay;
            proxy_pass http://signalling_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /api/users {
            limit_req zone=api burst=50 nodelay;
            proxy_pass http://users_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Static files
        location / {
            root /usr/share/nginx/html;
            try_files $uri $uri/ /index.html;
            
            # Caching for static assets
            location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
                expires 1y;
                add_header Cache-Control "public, immutable";
            }
        }

        # Health check
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
```

### TURN Server Configuration

```
# coturn.conf
listening-port=3478
tls-listening-port=5349
fingerprint
lt-cred-mech
use-auth-secret
static-auth-secret=your-turn-server-secret
realm=your-domain.com
total-quota=100
stale-nonce=600
cert=/etc/ssl/certs/your-domain.crt
pkey=/etc/ssl/private/your-domain.key
no-stdout-log
log-file=/var/log/coturn.log
pidfile=/var/run/turnserver.pid
```

## 📊 Monitoring & Observability

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'signalling-server'
    static_configs:
      - targets: ['signalling-server:8080']
    metrics_path: /metrics

  - job_name: 'users-service'
    static_configs:
      - targets: ['users-service:8081']
    metrics_path: /metrics

  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx:9113']

  - job_name: 'mongodb'
    static_configs:
      - targets: ['mongodb-exporter:9216']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "VideoConf Metrics",
    "panels": [
      {
        "title": "Active Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(websocket_connections_active)",
            "legendFormat": "Active WebSocket Connections"
          }
        ]
      },
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "5xx errors"
          }
        ]
      }
    ]
  }
}
```

## 🔐 Security Hardening

### SSL/TLS Configuration

```bash
# Generate SSL certificates with Let's Encrypt
certbot certonly --webroot -w /var/www/certbot -d your-domain.com

# Auto-renewal script
#!/bin/bash
certbot renew --quiet
docker-compose exec nginx nginx -s reload
```

### Security Headers

```nginx
# Add to nginx configuration
add_header X-Frame-Options DENY;
add_header X-Content-Type-Options nosniff;
add_header X-XSS-Protection "1; mode=block";
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";
add_header Content-Security-Policy "default-src 'self'; connect-src 'self' wss://your-domain.com; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';";
```

### Firewall Rules

```bash
# UFW firewall configuration
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw allow 3478/udp  # TURN
ufw allow 5349/tcp  # TURN/TLS
ufw deny incoming
ufw allow outgoing
ufw enable
```

## 📈 Scaling Guidelines

### Horizontal Scaling

1. **Application Servers**: Scale signalling servers based on WebSocket connections
   - Rule: 1 server per 1000 concurrent connections
   - Use session affinity for WebSocket connections

2. **Database**: Implement MongoDB replica sets
   - Primary for writes
   - Secondaries for reads
   - Automatic failover

3. **Load Balancing**: Use multiple load balancer instances
   - Configure health checks
   - Implement circuit breakers

### Vertical Scaling

```yaml
# Resource requirements per service
signalling-server:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

users-service:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

## 🚨 Disaster Recovery

### Backup Strategy

```bash
#!/bin/bash
# Daily backup script

# MongoDB backup
mongodump --uri="mongodb://user:pass@mongodb:27017/videoconf" --out="/backups/$(date +%Y%m%d)"

# Redis backup
redis-cli --rdb /backups/redis-$(date +%Y%m%d).rdb

# Upload to cloud storage
aws s3 sync /backups/ s3://your-backup-bucket/
```

### Recovery Procedures

1. **Database Recovery**:
   ```bash
   mongorestore --uri="mongodb://user:pass@mongodb:27017/videoconf" /backups/latest/
   ```

2. **Service Recovery**:
   ```bash
   kubectl rollout restart deployment/signalling-server -n videoconf
   kubectl rollout restart deployment/users-service -n videoconf
   ```

## 📋 Health Checks

### Application Health Endpoints

```go
// Health check endpoint implementation
func healthHandler(c *gin.Context) {
    // Check database connectivity
    if err := db.Ping(); err != nil {
        c.JSON(503, gin.H{
            "status": "unhealthy",
            "error": "database connection failed",
        })
        return
    }
    
    // Check Redis connectivity
    if err := redis.Ping(); err != nil {
        c.JSON(503, gin.H{
            "status": "unhealthy",
            "error": "redis connection failed",
        })
        return
    }
    
    c.JSON(200, gin.H{
        "status": "healthy",
        "timestamp": time.Now(),
        "version": "1.0.0",
    })
}
```

### Monitoring Scripts

```bash
#!/bin/bash
# Health monitoring script

services=("signalling-server:8080" "users-service:8081")

for service in "${services[@]}"; do
    if ! curl -f "http://$service/health" > /dev/null 2>&1; then
        echo "ALERT: $service is unhealthy"
        # Send alert (Slack, email, etc.)
    fi
done
```

## 🎯 Performance Tuning

### Database Optimization

```javascript
// MongoDB indexes
db.sessions.createIndex({ "host": 1, "createdAt": 1 })
db.users.createIndex({ "username": 1 }, { unique: true })
db.users.createIndex({ "email": 1 }, { unique: true })
```

### Go Service Optimization

```go
// Connection pooling
func init() {
    // Set MongoDB connection pool settings
    clientOptions := options.Client().ApplyURI(mongoURI).
        SetMaxPoolSize(50).
        SetMinPoolSize(5).
        SetMaxConnIdleTime(30 * time.Second)
}
```

### NGINX Optimization

```nginx
# Performance optimizations
worker_processes auto;
worker_connections 2048;

# Enable gzip compression
gzip on;
gzip_types text/plain text/css application/json application/javascript;

# Enable HTTP/2
listen 443 ssl http2;
```

## 📞 Support & Maintenance

### Log Management

```yaml
# ELK Stack for log aggregation
elasticsearch:
  image: elasticsearch:7.17.0
  environment:
    - discovery.type=single-node

logstash:
  image: logstash:7.17.0
  volumes:
    - ./logstash.conf:/usr/share/logstash/pipeline/logstash.conf

kibana:
  image: kibana:7.17.0
  ports:
    - "5601:5601"
```

### Update Strategy

```bash
#!/bin/bash
# Zero-downtime deployment script

# Build new images
docker build -t signalling-server:new ./server/signalling-server
docker build -t users-service:new ./server/users-service

# Rolling update
kubectl set image deployment/signalling-server signalling-server=signalling-server:new -n videoconf
kubectl rollout status deployment/signalling-server -n videoconf

kubectl set image deployment/users-service users-service=users-service:new -n videoconf
kubectl rollout status deployment/users-service -n videoconf
```

---

## 📈 Cost Optimization

### Resource Planning

| Component | Small Scale | Medium Scale | Large Scale |
|-----------|-------------|--------------|-------------|
| CPU Cores | 4 | 16 | 64+ |
| Memory | 8GB | 32GB | 128GB+ |
| Storage | 100GB | 500GB | 2TB+ |
| Bandwidth | 100Mbps | 1Gbps | 10Gbps+ |
| Monthly Cost | ~$200 | ~$800 | ~$3000+ |

### Cloud Provider Recommendations

1. **AWS**: Use EKS for Kubernetes, RDS for MongoDB, ElastiCache for Redis
2. **GCP**: Use GKE for Kubernetes, Cloud MongoDB Atlas, Memorystore for Redis
3. **Azure**: Use AKS for Kubernetes, Cosmos DB for MongoDB, Azure Cache for Redis

---

Built with ❤️ by the VideoConf Team