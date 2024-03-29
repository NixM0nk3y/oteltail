FROM public.ecr.aws/sam/build-provided.al2023:latest

#--platform=linux/arm64

COPY install-go.sh /tmp

RUN dnf install git -y \
    && /tmp/install-go.sh \
    && rm /tmp/install-go.sh

ENV PATH=/usr/local/goenv/shims:/usr/local/goenv/bin:/root/go/bin:/var/lang/bin:/usr/local/bin:/usr/bin/:/bin:/opt/bin:/usr/local/opt/sam-cli/bin