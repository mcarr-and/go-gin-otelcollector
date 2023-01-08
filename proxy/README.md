

This is an example Go-gin application that demonstrates nested spans. 

This uses the opentelemetry instrumented http client [otelhttp](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/net/http/otelhttp) 

This is a simple pass through service that calls the Album service.

## Prerequisites 
Cluster must have the following deployed
* Jaeger
* Opentelemetry-collector
* Album-Store

## TODO:
* Unit test with mocked out album-store serving different responses.

# Run 

## Local

[local run](Run-Local.md)

## Docker-Compose

[Docker-Compose](Run-Docker-Compose-Limited.md)

# K3d Run

[k3D install service ](Run-K3D-Install.md)