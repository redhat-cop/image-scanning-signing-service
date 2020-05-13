Image Signing Operator - Testing
========================================

## Local OCP E2E Tests (Centos)

There are local centos based E2E tests under the scripts folder. This will setup the nessesary namespace and resources then trigger an E2E test to validate a basic signing.

*What is tested*
* Verify a centos based signing image correctly signs an image from a remote repository
* Verify a centos based signing image correctly signs an image from an OCP image stream

> :warning: **Apply Global Resources**: Being logged into an OCP instance and having installed the global SCC and CRD resources shown in the [Install Operator](../README.md#install-crd-and-resources) section of the README are required before running this test.

```
$ sh ./scripts/centos-e2e-test.sh
```