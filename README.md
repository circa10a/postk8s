# 📬 postk8s

A simple kubernetes operator to manage physical mail via [mailform.io](https://www.mailform.io/)

![Build Status](https://github.com/circa10a/postk8s/workflows/deploy/badge.svg)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/circa10a/postk8s)

<img width="40%" src="docs/assets/mail-gopher.png" align="right"/>

- [postk8s](#postk8s)
  - [Example spec](#example-spec)
  - [Install](#Install)
    - [Kubectl](#kubectl)
  - [Development](#development)

### Example spec

```yaml
apiVersion: mailform.circa10a.github.io/v1alpha1
kind: Mail
metadata:
  annotations:
    # Optionally skip cancelling orders on delete
    mailform.circa10a.github.io/skip-cancellation-on-delete: false
  labels:
    app.kubernetes.io/name: postk8s
    app.kubernetes.io/managed-by: kustomize
  name: mail-sample
spec:
  message: "Hello, this is a test mail sent via PostK8s!"
  service: USPS_PRIORITY
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
> The `MAILFORM_API_TOKEN` environment variable will need to be updated in the `postk8s-controller-manager` deployment.

```console
$ kubectl apply -f https://raw.githubusercontent.com/circa10a/postk8s/main/dist/install.yaml
```

### Development

For local development, simply have your kubernetes context set for a cluster, clone, and run:

```console
export MAILFORM_API_TOKEN="<token>"
$ make local
```

#### Install a sample mail resource

```console
$ make sample
```
