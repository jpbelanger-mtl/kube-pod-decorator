FROM scratch

ADD dist /app

CMD ["/app/kube-pod-decorator"]