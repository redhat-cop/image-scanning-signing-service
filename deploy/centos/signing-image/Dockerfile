FROM centos:8

ARG OCP_VERSION
ENV OCP_VERSION ${OCP_VERSION:-3.10}

ADD bin/sign-image /usr/local/bin/
USER 0
# The curl install of JQ is required in order to bypass requiring the EPEL repository. The URL can be mirrored in disconnected environments.
RUN yum repolist > /dev/null && \
    curl -o jq -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 && \
    chmod +x ./jq && \
    cp jq /usr/bin && \
    curl -o oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/openshift-client-linux.tar.gz && \
    tar -xvf oc.tar.gz && \
    chmod +x ./oc && \
    cp oc /usr/bin && \
    yum clean all && \
    INSTALL_PKGS="podman" && \
    yum install -y --setopt=tsflags=nodocs $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all && \ 
    sed 's/#mount_program/mount_program/' /etc/containers/storage.conf -i