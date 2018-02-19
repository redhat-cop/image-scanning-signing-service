FROM registry.access.redhat.com/rhel7/rhel:7.4

MAINTAINER Andrew Block <ablock@redhat.com>

LABEL io.k8s.description="Platform for building and running Go applications" \
      io.k8s.display-name="Go Source-To-Image Builder" \
      io.openshift.tags="builder,go" \
      io.openshift.s2i.scripts-url="image:///usr/local/s2i" \
      io.openshift.s2i.destination="/tmp" 

ENV HOME=/opt/app-root \
    GOPATH=/opt/app-root/go \
    GOBIN=/opt/app-root/go/bin \
    PATH=$PATH:/opt/app-root/go/bin \
    STI_SCRIPTS_PATH=/usr/local/s2i

COPY s2i $STI_SCRIPTS_PATH

RUN yum repolist > /dev/null && \
    yum-config-manager --enable rhel-7-server-optional-rpms && \
    yum clean all && \
    INSTALL_PKGS="golang" && \
    yum install -y --setopt=tsflags=nodocs $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all && \
    useradd -u 1001 -r -g 0 -d ${HOME} -s /sbin/nologin -c "Default Application User" default && \
    mkdir -p ${GOPATH}/{bin,src} && \
    chown -R 1001:0 ${HOME} && \
    chmod -R g+rwX ${HOME}

WORKDIR ${HOME}

USER 1001

CMD ["/usr/local/s2i/usage"]