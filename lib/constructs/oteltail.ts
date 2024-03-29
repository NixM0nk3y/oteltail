import { Construct } from "constructs";
import { IVpc } from "aws-cdk-lib/aws-ec2";
import {
    ApplicationLoadBalancer,
    ListenerCondition,
    ListenerAction,
    DesyncMitigationMode,
    IpAddressType,
} from "aws-cdk-lib/aws-elasticloadbalancingv2";
import { Bucket, BlockPublicAccess } from "aws-cdk-lib/aws-s3";
import { CfnOutput, Duration } from "aws-cdk-lib";

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
            versioned: false,
            publicReadAccess: false,
            blockPublicAccess: BlockPublicAccess.BLOCK_ALL,
            eventBridgeEnabled: true,
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

        // 👇 add the ALB DNS as an Output
        new CfnOutput(this, "albDNS", {
            value: alb.loadBalancerDnsName,
        });
    }
}
