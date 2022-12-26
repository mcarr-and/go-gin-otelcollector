## 0. Expected tooling to run this project in K3D

1. Go
2. Docker 
3. Skaffold
4. K3D 
5. local changes to your `/etc/hosts` to use nginx-ingress with your  

```127.0.0.1	localhost k-dashboard.local jaeger.local otel-collector.local```

## 1. Create K3d Cluster

```bash
make k3d-cluster-create
#kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.10.1/cert-manager.yaml;
```

## 2. Start All Observability & Log Viewing Services
 
```bash
make skaffold-dev-k3d;
```

## 3. Start album-store Go/Gin Server with flags set

* `-namespace` kubernetes namespace 
* `-instance-name` kubernetes instance name (unique name when horizontal scaling)
* `-otel-location` can be changed from K3D-Nginx `otel-collector.local`

```bash
make local-start-k3d-grpc;
```

#### Note: the application will not start without the OpenTelemetry collector running

## 4. Run Some Tests

[Postman Collection](../test/Album-Store.postman_collection.json)

```bash
make local-test;
```

## 5. View the events in the different Services in K3D

[View Jaeger](http://jaeger.local:8070/search?limit=20&service=album-store)

[View K-Dashboard to see Kubernetes environment in a browser](http://k-dashboard:8070/)

TODO Prometheus 

## 6. Stop album-store server & Services  

### 1. Stop Server

`Ctr + C` in the terminal window where go is running. 

### 2. Stop Observability and Log Viewing Services

Ctr + C on the terminal window where you started `make skaffold-dev`

## 7. Delete K3D Cluster

```bash
make k3d-cluster-delete
```