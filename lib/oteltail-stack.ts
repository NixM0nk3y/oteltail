import * as cdk from "aws-cdk-lib";
import { Construct } from "constructs";
import { Vpc } from "aws-cdk-lib/aws-ec2";
import { OtelTail } from "./constructs/oteltail";
export interface OtelTailStackProps extends cdk.StackProps {
    readonly tenant: string;
    readonly environment: string;
    readonly product: string;
}

export class OtelTailStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: OtelTailStackProps) {
        super(scope, id, props);

        const myVpc = Vpc.fromLookup(this, "externalVpc", {
            isDefault: true,
        });

        new OtelTail(this, "otelTail", {
            tenant: props.tenant,
            environment: props.environment,
            product: props.product,
            vpc: myVpc,
        });
    }
}
