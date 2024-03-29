import * as cdk from "aws-cdk-lib";
import { Construct } from "constructs";

export interface OtelTailStackProps extends cdk.StackProps {
    readonly tenant: string;
    readonly environment: string;
    readonly product: string;
}

export class OtelTailStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: OtelTailStackProps) {
        super(scope, id, props);
    }
}
