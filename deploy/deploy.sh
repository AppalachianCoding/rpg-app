#!/bin/bash
set -x

filedir="$(dirname "$0")"
ACCT_ID="$(aws sts get-caller-identity --query Account --output text)"
if [[ -z "$AWS_REGION" ]]; then
  AWS_REGION="$(aws configure get region)"
fi
CF_BUCKET="$ACCT_ID-$AWS_REGION-cfbucket"
STACK_NAME="$1"
BACKEND_IMAGE="$STACK_NAME-backend:latest"
FRONTEND_IMAGE="$STACK_NAME-frontend:latest"
DOCKER_ENDPOINT="$ACCT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"
BACKEND_REPO="$DOCKER_ENDPOINT/$STACK_NAME-backend"
FRONTEND_REPO="$DOCKER_ENDPOINT/$STACK_NAME-frontend"
BACKENDIR="$filedir/../backend"
FRONTENDIR="$filedir/../frontend"


deploy_to_ecr() {
  aws ecr get-login-password |
    docker login --username AWS --password-stdin "$DOCKER_ENDPOINT"

  docker build -t "$BACKEND_IMAGE" "$BACKENDIR"
  docker tag "$BACKEND_IMAGE" "$BACKEND_REPO"

  docker build -t "$FRONTEND_IMAGE" "$FRONTENDIR"
  docker tag "$FRONTEND_IMAGE" "$FRONTEND_REPO"

  echo "Waiting for ECR repository to be created..."
  until aws ecr describe-repositories --repository-names "$STACK_NAME-backend" \
    --query "repositories[0].repositoryArn" --output text 2>/dev/null
  do
    sleep 5
    echo -n "."
  done
  docker push "$BACKEND_REPO"

  until aws ecr describe-repositories --repository-names "$STACK_NAME-frontend" \
    --query "repositories[0].repositoryArn" --output text 2>/dev/null
  do
    sleep 5
    echo -n "."
  done
  docker push "$FRONTEND_REPO"
}

FAIL=false
for envvar in ACCT_ID AWS_REGION STACK_NAME; do
  if [ -z "${!envvar}" ]; then
    echo "Error: $envvar is not set."
    FAIL=true
  fi
done
if $FAIL; then
  exit 1
fi

if ! aws s3 ls "s3://$CF_BUCKET" > /dev/null; then
  aws s3 mb "s3://$CF_BUCKET"
fi

for file in "$filedir"/*.yml; do
  if ! aws cloudformation validate-template --template-body "file://$file"; then
    echo "Error: Invalid CloudFormation template $file"
    exit 1
  fi
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

  deploy_to_ecr

  ECS_STACK_NAME=$(aws cloudformation describe-stack-resources --stack-name "$STACK_NAME" \
    --logical-resource-id Ecs \
    --query "StackResources[].PhysicalResourceId" \
    --output text |
    cut -d/ -f2)

  ECS_CLUSTER=$(aws cloudformation describe-stack-resources \
    --stack-name "$ECS_STACK_NAME" \
    --query "StackResources[?ResourceType=='AWS::ECS::Cluster'].PhysicalResourceId" \
    --output text)

  ECS_SERVICE=$(aws cloudformation describe-stack-resources \
    --stack-name "$ECS_STACK_NAME" \
    --query "StackResources[?ResourceType=='AWS::ECS::Service'].PhysicalResourceId" \
    --output text)

  if [[ -z "$ECS_CLUSTER" || -z "$ECS_SERVICE" ]]; then
    echo "Error: Could not determine ECS cluster or service from stack ${STACK_NAME}"
    exit 1
  fi

  echo "Updating ECS service ${ECS_SERVICE} in cluster ${ECS_CLUSTER}..."
  for service in $ECS_SERVICE; do
    aws ecs update-service \
      --cluster "$ECS_CLUSTER" \
      --service "$service" \
      --force-new-deployment
  done
  echo "Waiting for ECS service to stabilize..."
  for service in $ECS_SERVICE; do
    aws ecs wait services-stable --cluster "$ECS_CLUSTER" --services "$service"
    echo "Service $service is stable."
  done

  echo "Waiting for stack update to complete..."
  aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME"

else
  aws cloudformation create-stack \
    --stack-name "$STACK_NAME" \
    --template-body "file://$filedir/deploy.yml" \
    --capabilities CAPABILITY_NAMED_IAM \
    --parameters \
      ParameterKey=CloudformationBucket,ParameterValue="$CF_BUCKET/$STACK_NAME" \
    --tags \
    Key=Project,Value="$STACK_NAME" 2>&1

  deploy_to_ecr

  aws cloudformation wait stack-create-complete --stack-name "$STACK_NAME"
fi
