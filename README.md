Image Signing Operator
========================================

_This repository is currently undergoing active development. Functionality may be in flux_

Operator to support signing of images within OCP 4.x clusters [OpenShift Container Platform](https://www.openshift.com/container-platform/index.html)

### Build & Run Locally (OpenShift)

Run the following steps to run the operator locally. The operator will require `cluster-admin` permissions that can be applied using the resources provided in the deploy/ folder.

Pull in dependences
```
$ export GO111MODULE=on
$ go mod vendor
```

Create the expected namespace
```
$ oc new-project image-management
```

Select a distribution

UBI
```
$ DISTRO=ubi
```
Note: For UBI build to work you need to add a subscription entitlement key
```
oc create secret generic etc-pki-entitlement --from-file=entitlement.pem=/path/to/pem/file/{id}.pem --from-file=entitlement-key.pem=/path/to/pem/file/{id}.pem

```
https://docs.openshift.com/container-platform/4.3/builds/running-entitled-builds.html#builds-source-secrets-entitlements_running-entitled-builds


Centos
```
$ DISTRO=centos
```

Add crd and resources to cluster
```
$ oc apply -f deploy/crds/imagesigningrequests.cop.redhat.com_imagesigningrequests_crd.yaml
$ oc apply -f deploy/service_account.yaml
$ oc apply -f deploy/role.yaml
$ oc apply -f deploy/role_binding.yaml
$ oc apply -f deploy/scc.yaml
$ oc apply -f deploy/secret.yaml
$ oc apply -f deploy/${DISTRO}/image.yaml
$ oc apply -f deploy/${DISTRO}/sign_build.yaml
```

Build signing image (locally)
```
$ cd /deploy/${DISTRO}/signing-image
$ oc start-build image-sign-scan-base --from-dir=./ --follow
```

Login to the cluster via the Service Account above
```
$ TOKEN=$(oc sa get-token imagemanager)
$ oc login --token="${TOKEN}"
```

Run Operator-SDK
```
$ operator-sdk run --local --namespace="image-management" 
```