Kubernetes: AWS Autoscaler
==========================

A simple autoscaler for Kubernetes + AWS using the Deployments API.

## How it works

Using the K8s Deployment API we use the following formula:

**Total CPU / Memory requests** + **Add "extra" instances** = **Desired amount of instances**

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