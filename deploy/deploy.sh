#!/bin/bash

filedir="$(dirname "$0")"
ACCT_ID="$(aws sts get-caller-identity --query Account --output text)"
AWS_REGION="$(aws configure get region)"
CF_BUCKET="$ACCT_ID-$AWS_REGION-cfbucket"
STACK_NAME="$1"
BACKEND_IMAGE="$STACK_NAME-backend:latest"
FRONTEND_IMAGE="$STACK_NAME-frontend:lastest"
DOCKER_ENDPOINT="$ACCT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"
BACKEND_REPO="$DOCKER_ENDPOINT/$STACK_NAME-backend"
FRONTEND_REPO="$DOCKER_ENDPOINT/$STACK_NAME-frontend"
BACKENDIR="$filedir/../backend"
FRONTENDIR="$filedir/../frontend"

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

aws ecr get-login-password |
  docker login --username AWS --password-stdin "$DOCKER_ENDPOINT"
docker build -t "$STACK_NAME-backend" "$BACKENDIR"
docker tag "$BACKEND_IMAGE" "$BACKEND_REPO"
docker build -t "$STACK_NAME-frontend" "$FRONTENDIR"
docker tag "$FRONTEND_IMAGE" "$FRONTEND_REPO"


if aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME"
then
  docker push "$BACKEND_REPO"
  docker push "$FRONTEND_REPO"

  aws cloudformation update-stack \
    --stack-name "$STACK_NAME" \
    --template-body "file://$filedir/deploy.yml" \
    --capabilities CAPABILITY_NAMED_IAM \
    --parameters \
      ParameterKey=CloudformationBucket,ParameterValue="$CF_BUCKET/$STACK_NAME" \
    --tags \
      Key=Project,Value="$STACK_NAME" 2>&1

  aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME" 

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

  TASK_ARNS=$(aws ecs list-tasks \
    --cluster "$ECS_CLUSTER" \
    --service-name "$ECS_SERVICE" \
    --desired-status RUNNING \
    --query 'taskArns[0]' \
    --output text)

  if [[ -z "$TASK_ARNS" ]]; then
    echo "No running tasks found in cluster ${CLUSTER_NAME} for service ${SERVICE_NAME}."
    exit 1
  fi

  NEW_IMAGE_DIGEST=$(docker build --quiet .)
  for TASK_ARN in $TASK_ARNS; do
    ECS_IMAGE_DIGEST=$(aws ecs describe-tasks \
      --cluster "${ECS_CLUSTER}" \
      --tasks "${TASK_ARN}" \
      --query 'tasks[0].containers[0].imageDigest' \
      --output text)

    if [[ "${NEW_IMAGE_DIGEST}" != "${ECS_IMAGE_DIGEST}" ]]; then
      echo "Updating ECS service ${ECS_SERVICE} in cluster ${ECS_CLUSTER}..."
      aws ecs update-service \
        --cluster "$ECS_CLUSTER" \
        --service "$ECS_SERVICE" \
        --force-new-deployment
      break
    fi
  done

else
  aws cloudformation create-stack \
    --stack-name "$STACK_NAME" \
    --template-body "file://$filedir/deploy.yml" \
    --capabilities CAPABILITY_NAMED_IAM \
    --parameters \
      ParameterKey=CloudformationBucket,ParameterValue="$CF_BUCKET/$STACK_NAME" \
    --tags \
    Key=Project,Value="$STACK_NAME" 2>&1

  echo "Waiting for ECR repository to be created..."
  until aws ecr describe-repositories --repository-names "$STACK_NAME" \
    --query "repositories[0].repositoryArn" --output text 2>/dev/null
  do
    sleep 5
    echo -n "."
  done

  docker push "$BACKEND_REPO"
  docker push "$FRONTEND_REPO"

  aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME"
fi
