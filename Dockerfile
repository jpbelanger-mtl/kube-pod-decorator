FROM busybox:latest

ADD build/kube-pod-decorator-linux-amd64.tgz /app

CMD ["/app/kube-pod-decorator"]