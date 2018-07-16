FROM registry.access.redhat.com/rhel7/rhel

ARG OCP_VERSION=3.9

ADD bin/sign-image bin/scan-image /usr/local/bin/

RUN rpm -ihv https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm && \
    yum repolist > /dev/null && \
    yum clean all && \
    INSTALL_PKGS="docker atomic jq atomic-openshift-clients image-inspector tar" && \
    yum install -y --enablerepo=rhel-7-server-ose-${OCP_VERSION}-rpms --enablerepo=rhel-7-server-extras-rpms --setopt=tsflags=nodocs $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all
