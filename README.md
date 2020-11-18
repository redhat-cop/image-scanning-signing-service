Image Signing Operator
========================================

_This repository is currently undergoing active development. Functionality may be in flux_

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

## Registry Types
This operator supports a wide range of registry types when declaring an image to sign. The type and location of the image to sign are found within the `containerImage` attribute of the `ImageSigningRequest` CR.

### Container Repository
Traditional format for utalizing a remote container, either by specifying a tag or digest. These are of kind `ContainerRepository` under the `containerImage` attribute.

#### Tag
```
containerImage:
  kind: ContainerRepository
  name: quay.io/redhat-cop/image-scanning-signing-service:latest
```
#### Digest
```
containerImage:
  kind: ContainerRepository
  name: quay.io/redhat-cop/image-scanning-signing-service&sha256:a47ae897b964f1e543452c31a24bbd3d46ed5830f4a6d9992be97d0ce61ceb6b
```

### ImageStreamTag (OpenShift)
Specify an OCP `ImageStream` along with the corresponding tag of the desired image to sign. These are of kind `ImageStreamTag` under the `containerImage` attribute.

```
containerImage:
  kind: ImageStreamTag
  name: image-scanning-signing-service:latest
```

## Pull Secrets
A pull secret can be included in the `ImageSigningRequest` for when needing to access a private repository to sign images.

```
spec:
  containerImage:
    kind: ContainerRepository
    name: quay.io/redhat-cop/image-scanning-signing-service:latest
  pullSecret
    name: quay
```

### Creating Pull Secret (OpenShift)
There are two options to create the secret needed for accessing a private repository.

#### Existing Docker Config File
If using docker login locally you can use your existing config.json file to create a secret with your tokens needed for remote login. 

> :warning: **Security Risk**: This will upload the tokens for all remote repositories that you have logged into locally.

```
 oc secrets new <pull_secret_name> \
     .dockerconfigjson=path/to/.docker/config.json
```

#### Existing Docker Config File
Create a new secret by including your repository's credentials within the oc cli secrets command.

```
oc secrets new-dockercfg <pull_secret_name> \
    --docker-server=<registry_server> --docker-username=<user_name> \
    --docker-password=<password> --docker-email=<email>
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
  containerImage:
    kind: ImageStreamTag
    name: dotnet-example:latest
```

The above example can be applied to the cluster by running

``` $ oc apply -f deploy/examples/imagestreamtag.yaml ```

The signing pod will launch in the `image-management` namespace and handle the signing of the specified image. the `ImageSigningRequest` in the `dotnet-example` namespace will be updated and contain the name of the signed image in the Status section. Confirm this by running 

``` $ oc get imagesigningrequest/dotnet-app -o yaml ```

Finally, the newly created Image will contain the signatures associated with the signing action. This can be confirmed by running the following command:

```
$ oc get image $(oc get imagesigningrequest dotnet-app --template='{{ .status.signedImage }}') -o yaml
```

## Development
### [How-To](docs/development.md)
### [Testing](docs/testing.md)
