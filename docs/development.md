Image Signing Operator - Development
========================================

## Build & Run Locally (OpenShift)

Run the following steps to run the operator locally. The operator will require `cluster-admin` permissions that can be applied using the resources provided in the deploy/ folder from the Install section above.

Pull in dependences
```
$ export GO111MODULE=on
$ go mod vendor
```

### Select a distribution

### UBI
```
$ DISTRO=ubi
$ oc apply -f deploy/${DISTRO}/image.yaml
```
Note: For UBI signing image build to work you need to add a subscription entitlement key
```
$ oc create secret generic etc-pki-entitlement --from-file=entitlement.pem=path/to/pem/file/{id}.pem --from-file=entitlement-key.pem=path/to/pem/file/{id}.pem

```
*Additional reading on entitled builds*
https://docs.openshift.com/container-platform/4.3/builds/running-entitled-builds.html#builds-source-secrets-entitlements_running-entitled-builds


### Centos
```
$ DISTRO=centos
$ oc apply -f deploy/${DISTRO}/image.yaml
```

### Build Signing Image GIT
Build signing image from remote GIT repository
```
$ oc apply -f deploy/${DISTRO}/sign_build.yaml
$ oc start-build image-sign-scan-base --follow
```

### Build Signing Image Locally
Build signing image locally 
```
$ oc apply -f deploy/${DISTRO}/sign_build_local.yaml
$ oc start-build image-sign-scan-base --from-dir=./deploy/${DISTRO}/signing-image --follow
```

### Run Operator-SDK
Login to the cluster via the Service Account above
```
$ TOKEN=$(oc sa get-token imagemanager)
$ oc login --token="${TOKEN}"
```
Run the operator locally
```
$ operator-sdk run --local --namespace="image-management" 
```

## [Testing](testing.md)