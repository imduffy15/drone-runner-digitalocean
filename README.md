# drone-runner-gcp

The gcp runner provisions and executes pipelines on a Google Cloud Platform instance using the ssh protocol. A new instance is provisioned for each pipeline execution, and then destroyed when the pipeline completes. This runner is intended for workloads that are not suitable for running inside containers. Drone server 1.4.0 or higher is required.
