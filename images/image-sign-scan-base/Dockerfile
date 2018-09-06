FROM registry.access.redhat.com/openshift3/image-inspector:latest

ARG OCP_VERSION=3.9

ADD bin/sign-image bin/scan-image /usr/local/bin/

RUN yum repolist > /dev/null && \
    curl -o jq -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 && \
    chmod +x ./jq && \
    cp jq /usr/bin && \
    yum clean all && \
    INSTALL_PKGS="docker atomic atomic-openshift-clients tar" && \
    yum install -y --enablerepo=rhel-7-server-ose-${OCP_VERSION}-rpms --enablerepo=rhel-7-server-extras-rpms --setopt=tsflags=nodocs $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all
