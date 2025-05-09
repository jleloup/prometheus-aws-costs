# prometheus-aws-costs

Prometheus exporter exposing AWS Cost Explorer &amp; Billing metrics

## Deployment

You can use the Helm chart located in `charts/` to deploy the Prometheus AWS Cost exporter in your Kubernetes cluster.

## Cost impact

Official documentation for AWS Cost Explorer [pricing](https://aws.amazon.com/aws-cost-management/aws-cost-explorer/pricing/) indicates that each API query cost $0.01.

It is recommended to not go below 1h for the metric internal configuration for costs reasons and since AWS Cost Explorer does update hourly anyway.

## Contributing

This repository provides Skaffold configuration to build & run on a local Minikube cluster for development purposes.
