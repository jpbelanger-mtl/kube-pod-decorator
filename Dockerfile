FROM busybox:latest

ADD dist /app

CMD ["/app/kube-pod-decorator"]