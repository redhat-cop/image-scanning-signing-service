Image Signing Operator
========================================

_This repository is currently undergoing active development. Functionality may be in flux_

Operator to support signing of images within OCP 4.x clusters [OpenShift Container Platform](https://www.openshift.com/container-platform/index.html)

### Run Locally (OpenShift)

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

Add crd and resources to cluster
```
$ oc apply -f deploy/crds/imagesigningrequests.cop.redhat.com_imagesigningrequests_crd.yaml
$ oc apply -f deploy/service_account.yaml
$ oc apply -f deploy/role.yaml
$ oc apply -f deploy/role_binding.yaml
$ oc apply -f deploy/scc.yaml
$ oc apply -f deploy/secret.yaml
$ oc apply -f deploy/images.yaml
$ oc apply -f deploy/sign_build.yaml
```

Login to the cluster via the Service Account above
```
$ oc sa get-token imagemanager
$ oc login --token="{above_token}"
```

Run Operator-SDK
```
$ operator-sdk up local --namespace="image-management" 
```