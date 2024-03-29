import { Construct } from "constructs";
import { Vpc } from "aws-cdk-lib/aws-ec2";

export interface OtelTailProps {
    readonly tenant: string;
    readonly environment: string;
    readonly product: string;
    readonly vpc: Vpc;
}

export class OtelTail extends Construct {
    constructor(scope: Construct, id: string, props: OtelTailProps) {
        super(scope, id);

    }
}
