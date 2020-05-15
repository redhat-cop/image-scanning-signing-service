#!/bin/bash
# Run from projects root folder

# NOTE: You must have have run the following oc command for this to work. It is not automated 
# since it is a global resource
# $ oc apply -f deploy/scc.yaml

# Warning that there is an oc delete within this script
read -p "Warning - This script uses oc delete, use only on temporary instances; Press enter to continue or ctrl-c to exit"
# Setup the test namespace and required resources
oc new-project image-management-test
oc apply -f test/e2e/deploy/service_account.yaml
oc apply -f test/e2e/deploy/role.yaml
oc apply -f test/e2e/deploy/role_binding.yaml
oc apply -f test/e2e/deploy/secret.yaml
oc apply -f test/e2e/deploy/operator.yaml
oc rollout status deployment image-security --watch

# Create namespace that will hold the image to be signed
# The E2E test will fail if this namespace and image are not present on the system
oc new-project signing-test
oc new-app --template=dotnet-example
# Wait till the image is availble before testing against it
oc rollout status deploymentconfig dotnet-example --watch

# Run the E2E test
operator-sdk test local ./test/e2e/centos --namespace "image-management-test" --no-setup

# Remove testing namespaces
# oc delete project/signing-test
# oc delete project/image-management-test