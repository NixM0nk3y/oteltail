# Welcome to oteltail

Example of streaming S3 logs into a otel stack using a modified lambda based on a modified [promtail](https://github.com/grafana/loki/tree/main/tools/lambda-promtail/lambda-promtail) lambda

At the time of writing opentelemetry go sdk logging support is still in [progress](https://github.com/open-telemetry/opentelemetry-go/pull/4955/files#) so I've borrow heavily from a [seperate PR](https://github.com/dagger/dagger/pull/6835) so provide a functional otel logging SDK.

## Useful commands

-   `make clean` remove any intermediate state
-   `make diff` compare deployed stack with current state
-   `make deploy ` deploy this stack to your default AWS account/region
