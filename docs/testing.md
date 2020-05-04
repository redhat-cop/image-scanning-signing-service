Image Signing Operator - Testing
========================================

## Local E2E Test (Centos)

There is a local centos based E2E test under the scripts folder. This will setup the nessesary namespace and resources then trigger an E2E test to validate a basic signing.

> :warning: **Apply Global Resources**: Being logged into an OCP instance and having installed the global SCC and CRD resources shown in the [Install Operator](../README.md#install-crd-and-resources) section of the README are required before running this test.

```
$ sh ./scripts/centos-e2e-test.sh
```