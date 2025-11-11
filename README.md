# 📬 postk8s

A simple kubernetes operator to manage physical mail via [mailform.io](https://www.mailform.io/)

![Build Status](https://github.com/circa10a/postk8s/workflows/deploy/badge.svg)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/circa10a/postk8s)

<img width="40%" src="docs/assets/mail-gopher.png" align="right"/>

- [postk8s](#postk8s)
  - [Example spec](#example-spec)
  - [Install](#Install)
    - [Kubectl](#kubectl)
  - [Configuration Options](#configuration-options)
  - [Development](#development)

### Example spec

```yaml
apiVersion: mailform.circa10a.github.io/v1alpha1
kind: Mail
metadata:
  name: mail-sample
  annotations:
    # Optionally skip cancelling orders on delete
    mailform.circa10a.github.io/skip-cancellation-on-delete: false
spec:
  message: "Hello, this is a test mail sent via PostK8s!"
  service: USPS_STANDARD
  url: https://pdfobject.com/pdf/sample.pdf
  from:
    address1: 123 Sender St
    address2: Suite 100
    city: Senderville
    country: US
    name: Sender Name
    organization: Acme Sender
    postcode: "94016"
    state: CA
  to:
    address1: 456 Recipient Ave
    address2: Apt 4B
    city: Receivertown
    country: US
    name: Recipient Name
    organization: Acme Recipient
    postcode: "10001"
    state: NY
```

### Install

#### Kubectl

> [!IMPORTANT]
> The `MAILFORM_API_TOKEN` environment variable will need to be updated in the `postk8s-controller-manager` deployment in the `postk8s-system` namespace.

```console
kubectl apply -f https://raw.githubusercontent.com/circa10a/postk8s/main/dist/install.yaml
```

### Configuration options

```console
  -enable-http2
        If set, HTTP/2 will be enabled for the metrics and webhook servers
  -health-probe-bind-address string
        The address the probe endpoint binds to. (default ":8081")
  -kubeconfig string
        Paths to a kubeconfig. Only required if out-of-cluster.
  -leader-elect
        Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
  -mailform-api-token string
        Mailform API token.Defaults to 'MAILFORM_API_TOKEN' environment variable. (default "")
  -metrics-bind-address string
        The address the metrics endpoint binds to. Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service. (default "0")
  -metrics-cert-key string
        The name of the metrics server key file. (default "tls.key")
  -metrics-cert-name string
        The name of the metrics server certificate file. (default "tls.crt")
  -metrics-cert-path string
        The directory that contains the metrics server certificate.
  -metrics-secure
        If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead. (default true)
  -sync-interval string
        Interval to check for mail updates.Defaults to '12h'. (default "12h")
  -webhook-cert-key string
        The name of the webhook key file. (default "tls.key")
  -webhook-cert-name string
        The name of the webhook certificate file. (default "tls.crt")
  -webhook-cert-path string
        The directory that contains the webhook certificate.
  -zap-devel
        Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default true)
  -zap-encoder value
        Zap log encoding (one of 'json' or 'console')
  -zap-log-level value
        Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', 'panic'or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
        Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
  -zap-time-encoding value
        Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
```

### Development

For local development, simply have your kubernetes context set for a cluster, clone, and run:

```console
export MAILFORM_API_TOKEN="<token>"
make local
```

#### Install a sample mail resource

```console
make sample
```
