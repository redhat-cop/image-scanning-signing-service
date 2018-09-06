FROM registry.access.redhat.com/openshift3/image-inspector:latest

ARG OCP_VERSION=3.9

ADD bin/sign-image bin/scan-image /usr/local/bin/

RUN yum repolist > /dev/null && \
    yum clean all && \
    INSTALL_PKGS="docker atomic jq atomic-openshift-clients tar" && \
    yum install -y --enablerepo=rhel-7-server-ose-${OCP_VERSION}-rpms --enablerepo=rhel-7-server-extras-rpms --setopt=tsflags=nodocs $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all
