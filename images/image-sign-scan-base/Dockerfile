FROM openshift3/image-inspector:latest

ARG OCP_VERSION
ENV OCP_VERSION ${OCP_VERSION:-3.10}

ADD bin/sign-image bin/scan-image /usr/local/bin/

# The curl install of JQ is required in order to bypass requiring the EPEL repository. The URL can be mirrored in disconnected environments.
RUN yum repolist > /dev/null && \
    curl -o jq -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 && \
    chmod +x ./jq && \
    cp jq /usr/bin && \
    yum clean all && \
    INSTALL_PKGS="docker atomic atomic-openshift-clients tar" && \
    yum install -y --enablerepo=rhel-7-server-ose-${OCP_VERSION}-rpms --enablerepo=rhel-7-server-extras-rpms --setopt=tsflags=nodocs $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all
