# Integrating OpenTelemetry (OTEL) with Your Application

## Overview

OpenTelemetry (OTEL) provides a set of tools, APIs, and SDKs to
instrument, generate, collect, and export telemetry data (metrics, logs, and traces)
for monitoring and troubleshooting your applications. This guide will help you integrate
OTEL into your application, enabling you to capture and export metrics and traces.

## Setting Up the OTEL Collector Service on GCP

This guide provides step-by-step instructions to set up the
OpenTelemetry (OTEL) Collector Service on
Google Cloud Platform (GCP) using the provided configuration file.

## Prerequisites

Before enabling OTEL in your application, ensure that
the collector service is up and running.
The collector service is responsible for capturing
and exporting the telemetry data.
By default, OTEL is disabled in the configuration.

*   **GCP Account**: Ensure you have a GCP account and have access to a GCP project.
*   **Google Cloud SDK**: Install the [Google Cloud SDK](https://cloud.google.com/sdk/docs/install).
*   **Docker**: Install Docker to run the OTEL Collector in a container.
*   **Kubernetes Cluster (GKE)**: Set up a Google Kubernetes Engine (GKE) cluster
    if deploying in a Kubernetes environment.

## Steps to Set Up and Configure the OTEL Collector Service with Proxy Adaptor

### Step 1

Use the below `config.yaml` file (or set up the
`CONFIG_FILE` environment variable) for OTEL collector service

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
      http:
        endpoint: "0.0.0.0:55681"

processors:
  batch:
    send_batch_max_size: 0
    send_batch_size: 8192
    timeout: 5s

  memory_limiter:
    check_interval: 5s
    limit_percentage: 65
    spike_limit_percentage: 20

  resourcedetection:
    detectors: [gcp]
    timeout: 10s

exporters:
  googlecloud:
    metric:
      instrumentation_library_labels: true
      service_resource_labels: true

extensions:
  health_check:
    endpoint: "0.0.0.0:13133"

service:
  extensions: [health_check]
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch, memory_limiter, resourcedetection]
      exporters: [googlecloud]
    traces:
      receivers: [otlp]
      processors: [batch, memory_limiter, resourcedetection]
      exporters: [googlecloud]
```

### Step 2

Configure Proxy Adaptor

Follow these steps to enable and configure OTEL in your application:

1.  **Edit the `config.yaml` (or set up the `CONFIG_FILE` env) file:**
  *   Set the `enabled` field to `true` to enable metrics and traces.
  *   Configure the endpoints for the collector service to export the
  metrics and traces.
  *   Set the health check endpoint, which is configured in the collector service.
      Avoid including `http://` in the endpoints.
      Refer to the `example_config.yaml` file for guidance.

2.  **Example Configuration Block for OTEL:**

```yaml
   otel:
     enabled: true
     serviceName: YOUR_OTEL_COLLECTOR_SERVICE_NAME
     healthcheck:
       enabled: false
       endpoint: YOUR_OTEL_COLLECTOR_HEALTHCHECK_ENDPOINT
     metrics:
       endpoint: YOUR_OTEL_COLLECTOR_SERVICE_ENDPOINT
     traces:
       endpoint: YOUR_OTEL_COLLECTOR_SERVICE_ENDPOINT
       samplingRatio: YOUR_SAMPLING_RATIO
```

### Step 3: Set Up Proxy Adaptor & OTEL Collector Service as a Sidecar on GKE

*   Use the `proxy-adapter-application-as-sidecar.yaml` file from the `deployment/sidecar-k8`
    to set up the OTEL collector service along with the proxy adaptor as a sidecar.
*   You can find step-by-step instructions in the
    `/deployment/sidecar-k8/README.md` file.

### Step 4: Verify OTEL Changes

1.  **Verify the Health Check:**

  Run a curl command from the Kubernetes pod to `http://collector-ip:13133/health_check`.
  You should see a health status message.

1.  **Check OTEL Traces on GCP:**

*   Log in to your GCP project and open the Monitoring service.
*   Open the sidebar on the left and click on **Trace Explorer**.
*   You should be able to see the traces as shown below:

  ![Traces Execute](./img/traces-execute.png)
  ![Traces Batch](./img/traces-batch.png)

1.  **Check OTEL Metrics on GCP:**

*   Log in to your GCP project and open the Monitoring service.
*   Open the sidebar on the left and click on **Metrics Explorer**.
*   You should be able to see the categories as shown below:

    ![Metrics Category](./img/metrics-category.png)

*   Select options **Prometheus Target/Spanner**.
*   To view the metrics for the total number of requests, select `prometheus/spanner_dynamodb_adapter_<dbname>_request_count_total/counter`.

    ![Metrics Total Requests](./img/metrics_total_requests.png)

*   To view the metrics for latency, select `prometheus/spanner_dynamodb_adapter_round_trip_latencies_milliseconds/histogram`.

    ![Metrics Latency](./img/metrics-latency.png)

*   You can also view other metrics related to the
    Spanner library under the **Prometheus Target/Spanner** category.
