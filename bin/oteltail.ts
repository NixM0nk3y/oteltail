#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { GossipyKeyValStack } from '../lib/oteltail-stack';

function capitalise(s: string): string {
    return s[0].toUpperCase() + s.substr(1).toLowerCase();
}

const app = new cdk.App();

const tenant = process.env.TENANT ?? "Abc";
const product = process.env.PRODUCT ?? "OtelTail";
const environment = process.env.ENVIRONMENT ?? "Dev";

new OtelTailStack(app, capitalise(tenant) + capitalise(product) + capitalise(environment), {
    tenant: tenant,
    environment: environment,
    product: product,
    env: { account: process.env.CDK_DEFAULT_ACCOUNT, region: process.env.CDK_DEFAULT_REGION ?? "eu-west-1" },
});