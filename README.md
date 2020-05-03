Image Signing Operator
========================================

_This repository is currently undergoing active development. Functionality may be in flux_

Operator to support signing of images within OCP 4.x clusters [OpenShift Container Platform](https://www.openshift.com/container-platform/index.html)

## Install Operator

### Create Namespace
```
$ oc new-project image-management
```

### Install CRD and Resources
```
$ oc apply -f deploy/crds/imagesigningrequests.cop.redhat.com_imagesigningrequests_crd.yaml
$ oc apply -f deploy/service_account.yaml
$ oc apply -f deploy/role.yaml
$ oc apply -f deploy/role_binding.yaml
$ oc apply -f deploy/scc.yaml
$ oc apply -f deploy/secret.yaml
```

### Deploy 
Apply the operator to the image-management namespace
```
$ oc apply -f deploy/operator.yaml
```

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

## Example Workflow (OpenShift)

To facilitate Image Signing, the image signer makes use of a `ImageSigningRequest` Custom Resource Definition which allows users to declare their intent to have an image signed. This section will walk through the process of signing an image after a new image has been built.

OpenShift provides a number of quickstart templates. One of these templates contains a simple .NET Core web application application. This is an ideal use case to showcase image signing in action.
Build an Application

First, create a new project called dotnet-example

```$ oc new-project dotnet-example```

Instantiate the dotnet-example template within the project using the default values specified in the template

```$ oc new-app --template=dotnet-example```

### Declare an Intent to Sign the Image

To declare your intent to sign the previously built image, a new `ImageSigningRequest` can be created within the project. A typical request is shown below

```
apiVersion: imagesigningrequests.cop.redhat.com/v1alpha1
kind: ImageSigningRequest
metadata:
  name: dotnet-app
spec:
  imageStreamTag: dotnet-example:latest
```

The above example can be applied to the cluster by running

``` $ oc apply -f deploy/examples/example.yaml ```

The signing pod will launch in the `image-management` namespace and handle the signing of the specified image. the `ImageSigningRequest` in the `dotnet-example` namespace will be updated and contain the name of the signed image in the Status section. Confirm this by running 

``` $ oc get imagesigningrequest/dotnet-app -o yaml ```

Finally, the newly created Image will contain the signatures associated with the signing action. This can be confirmed by running the following command:

```
$ oc get image $(oc get imagesigningrequest dotnet-app --template='{{ .status.signedImage }}') -o yaml
```
