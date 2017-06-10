Kubernetes: AWS Autoscaler
==========================

A simple autoscaler for Kubernetes + AWS using the Deployments API.

## How it works

Using the K8s Deployment API we have the following formula:

```
CPU / Memory requests + "extra" instances = Desired amount of instances
```

The "extra" instances allow for buffer when running dynamically provisioned pods eg. Jobs.

## Development

**Build the binary**

```bash
make build
```

**Build a Docker container**

```bash
make image
```