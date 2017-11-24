FROM busybox

COPY ./instrumented_app /instrumented_app

EXPOSE 8080

ENTRYPOINT [ "/instrumented_app" ]
CMD []
