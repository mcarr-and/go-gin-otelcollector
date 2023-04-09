# grafana-dashboards

![Version: 1.0.2](https://img.shields.io/badge/Version-1.0.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.0.0](https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square)

A Helm chart that deploys my collection of Grafana dashboards

**Homepage:** <https://github.com/mcarr-and/go-gin-otelcollector>

## Source Code

* <https://github.com/mcarr-and/go-gin-otelcollector>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| dashboards.golang | bool | `true` | Toggles the crowdsec dashboards |
| dashboards.jaegerAllInOne | bool | `true` | Toggles the cert-manager dashboards |
| dashboards.kafka | bool | `true` | Toggles the kafka dashboards |
| dashboards.otelCollector | bool | `true` | Toggles the otelCollector dashboards |
| dashboards.prometheus | bool | `true` | Toggles the cert-manager dashboards |
| dashboards.zookeeper | bool | `true` | Toggles the zookeeper dashboards |
| fullnameOverride | string | `""` | String to fully override names.fullname |
| nameOverride | string | `""` | String to partially override names.fullname |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.11.0](https://github.com/norwoodj/helm-docs/releases/v1.11.0)