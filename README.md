Kubernetes: AWS Autoscaler
==========================

A simple autoscaler for Kubernetes + AWS using the Deployments API.

## How it works

Using the K8s Deployment API we have the following formula:

![Diagram](/docs/diagram.png "Diagram")

The "extra" instances allow for buffer when running dynamically provisioned pods eg. Jobs.

## Development

**Run the tests**

```bash
make lint
make test
```

**Build the binary**

```bash
make build
```

**Build a Docker container**

```bash
make image
```
