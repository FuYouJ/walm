#build
FROM 172.16.1.99/transwarp/gcr.io/google-containers/kube-cross:v1.12.1-2 AS build-env
ADD . /go/src/helm.sh/helm
RUN cd /go/src/helm.sh/helm && make build

#runner
FROM 172.16.1.99/transwarp/alpine:transwarp-base
COPY --from=build-env /go/src/helm.sh/helm/bin/helm /usr/bin/
RUN chmod +x /usr/bin/helm
