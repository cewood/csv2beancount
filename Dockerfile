FROM scratch

ARG TARGETOS
ARG TARGETARCH
ARG TARGETPLATFORM
ARG TARGETVARIANT

COPY dist/csv2beancount_${TARGETOS}_${TARGETARCH}/csv2beancount /

ARG CREATED
ARG REVISION=HEAD

LABEL org.opencontainers.image.authors="https://github.com/cewood" \
      org.opencontainers.image.created="${CREATED}" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.source="https://github.com/cewood/csv2beancount/tree/${REVISION}" \
      org.opencontainers.image.title="cewood/csv2beancount" \
      org.opencontainers.image.url="https://github.com/cewood/csv2beancount"
