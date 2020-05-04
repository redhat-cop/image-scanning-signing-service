Image Signing Operator - Testing
========================================

## Local E2E Test (Centos)

There is a local centos based E2E test under the scripts folder. This will setup the nessesary namespace and resources then trigger an E2E test to validate a basic signing.

*Make sure to run this from the root project folder*
```
$ sh ./scripts/centos-e2e-test.sh
```