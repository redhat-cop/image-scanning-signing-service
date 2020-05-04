Image Signing Operator - Testing
========================================

## Local E2E Test (Centos)

There is a local centos based E2E test under the scripts folder. This will setup the nessesary namespace and resources then trigger an E2E test to validate a basic signing.

> :warning: **Apply SCC first**: Applying the global SCC and CRD resources shown in the [install](../README.md) is required before running this test

*Make sure to run this from the root project folder*
```
$ sh ./scripts/centos-e2e-test.sh
```