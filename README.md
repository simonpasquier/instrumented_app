This is an example of application instrumented for Prometheus. By default it
listens on port 8080 and exposes metrics on the `/metrics` endpoint.

The root endpoint will respond with a random delay (1 second at most).

Exposed metrics:

* `orders_total`, a fast moving counter.
* `order_errors_total{stage=<stage>}`, a slow moving counter.
* `user_sessions`, a gauge value randomly set between 0 and 100.

## Usage

```
$ ./instrumented_app --help
usage: instrumented_app [<flags>]

Flags:
  --help                     Show context-sensitive help (also try --help-long and --help-man).
  --listen="127.0.0.1:8080"  Listen address
  --listen-metrics=""        Listen address for exposing metrics (default to 'listen' if blank)
  --basic-auth=""            Basic authentication (eg <user>:<password>)

```

