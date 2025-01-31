#!/bin/bash
set -x

filedir="$(dirname "$0")"
ACCT_ID="$(aws sts get-caller-identity --query Account --output text)"
AWS_REGION="$(aws configure get region)"
CF_BUCKET="$ACCT_ID-$AWS_REGION-cfbucket"
STACK_NAME="$1"
if [ -z "$STACK_NAME" ]; then
  echo "Usage: $0 <stack-name>"
  exit 1
fi

if ! aws s3 ls "s3://$CF_BUCKET" > /dev/null; then
  aws s3 mb "s3://$CF_BUCKET"
fi

for file in "$filedir"/*.yml; do
  aws s3 cp "$file" "s3://$CF_BUCKET/$STACK_NAME/$(basename "$file")"
done

if aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME"
then
  aws cloudformation update-stack \
    --stack-name "$STACK_NAME" \
    --template-body "file://$filedir/deploy.yml" \
    --capabilities CAPABILITY_NAMED_IAM \
    --parameters \
      ParameterKey=CloudformationBucket,ParameterValue="$CF_BUCKET/$STACK_NAME" \
    --tags \
      Key=Project,Value="$STACK_NAME" 2>&1
  aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME" 
else
  aws cloudformation create-stack \
    --stack-name "$STACK_NAME" \
    --template-body "file://$file_dir/deploy.yml" \
    --capabilities CAPABILITY_NAMED_IAM \
    --parameters \
      ParameterKey=CloudformationBucket,ParameterValue="$CF_BUCKET/$STACK_NAME" \
    --tags \
    Key=Project,Value="$STACK_NAME" 2>&1
  aws cloudformation wait stack-create-complete --stack-name "$STACK_NAME" 
fi
