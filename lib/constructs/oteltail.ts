import { Construct } from "constructs";
import { IVpc } from "aws-cdk-lib/aws-ec2";
import {
    ApplicationLoadBalancer,
    ListenerCondition,
    ListenerAction,
    DesyncMitigationMode,
    IpAddressType,
} from "aws-cdk-lib/aws-elasticloadbalancingv2";
import { Bucket, BlockPublicAccess, StorageClass } from "aws-cdk-lib/aws-s3";
import { CfnOutput, Duration, DockerImage, RemovalPolicy } from "aws-cdk-lib";
import { Function, Runtime, Code, Architecture, Tracing } from "aws-cdk-lib/aws-lambda";
import { RetentionDays } from "aws-cdk-lib/aws-logs";
import { Rule } from "aws-cdk-lib/aws-events";
import { LambdaFunction } from "aws-cdk-lib/aws-events-targets";

export interface OtelTailProps {
    readonly tenant: string;
    readonly environment: string;
    readonly product: string;
    readonly vpc: IVpc;
}

export class OtelTail extends Construct {
    constructor(scope: Construct, id: string, props: OtelTailProps) {
        super(scope, id);

        const loggingBucket = new Bucket(this, "albLoggingBucket", {
            removalPolicy: RemovalPolicy.DESTROY,
            versioned: false,
            publicReadAccess: false,
            blockPublicAccess: BlockPublicAccess.BLOCK_ALL,
            eventBridgeEnabled: true,
            lifecycleRules: [
                {
                    abortIncompleteMultipartUploadAfter: Duration.days(7),
                    expiration: Duration.days(90),
                    transitions: [
                        {
                            storageClass: StorageClass.INFREQUENT_ACCESS,
                            transitionAfter: Duration.days(30),
                        },
                    ],
                },
            ],
        });

        const alb = new ApplicationLoadBalancer(this, "alb", {
            vpc: props.vpc,
            internetFacing: true,
            http2Enabled: true,
            idleTimeout: Duration.seconds(1000),
            dropInvalidHeaderFields: true,
            desyncMitigationMode: DesyncMitigationMode.DEFENSIVE,
            ipAddressType: IpAddressType.IPV4,
            clientKeepAlive: Duration.seconds(500),
            crossZoneEnabled: true,
        });

        alb.logAccessLogs(loggingBucket, "albLogs");

        const listener = alb.addListener("Listener", {
            port: 80,
            open: true,
        });

        listener.connections.allowDefaultPortFromAnyIpv4("Open to the world");

        listener.addAction("Ok", {
            priority: 10,
            conditions: [ListenerCondition.pathPatterns(["/ok"])],
            action: ListenerAction.fixedResponse(200, {
                contentType: "text/plain",
                messageBody: "OK",
            }),
        });

        listener.addAction("Default", {
            action: ListenerAction.fixedResponse(404, {
                contentType: "text/plain",
                messageBody: "NOT_FOUND",
            }),
        });

        const otelTailFunction = new Function(this, "Lambda", {
            runtime: Runtime.PROVIDED_AL2023,
            handler: "bootstrap",
            architecture: Architecture.ARM_64,
            code: Code.fromAsset("./resources/lambda-promtail-otel", {
                bundling: {
                    image: DockerImage.fromBuild("./resources/buildimage"),
                    command: [...["make", "build-Lambda"]],
                    environment: {
                        HOME: "/tmp",
                        ARTIFACTS_DIR: "/asset-output",
                    },
                },
            }),
            environment: {
                LOG_LEVEL: "INFO",
                PRINT_LOG_LINE: "true",
                WRITE_ADDRESS: "https://httpbin.org/post",
            },
            tracing: Tracing.ACTIVE,
            logRetention: RetentionDays.ONE_WEEK,
            memorySize: 512,
            timeout: Duration.seconds(300),
        });

        loggingBucket.grantRead(otelTailFunction);

        const objectCreatedRule = new Rule(this, "objectCreatedRule", {
            description: "object created",
            eventPattern: {
                source: ["aws.s3"],
                detailType: ["Object Created"],
                detail: {
                    bucket: {
                        name: [loggingBucket.bucketName],
                    },
                },
            },
        });

        objectCreatedRule.addTarget(new LambdaFunction(otelTailFunction));

        new CfnOutput(this, "albDNS", {
            value: alb.loadBalancerDnsName,
        });
    }
}
