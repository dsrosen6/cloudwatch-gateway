#!/usr/bin/env bash

FUNCTION_NAME="cloudwatch-gateway"

export AWS_PAGER=""

build_and_zip() {
    if ! GOOS=linux GOARCH=amd64 go build -o bootstrap >/dev/null; then
        echo "error building binary"
        return 1
    fi

    if ! zip function.zip bootstrap >/dev/null; then
        echo "error zipping binary"
        return 1
    fi

    return 0
}

create_function() {
    FUNCTION_NAME=$1
    IAM_ROLE_ARN=$2
    if [[ -z "$FUNCTION_NAME" ]]; then
        echo "error: function name not provided"
        return 1
    fi

    if [[ -z "$IAM_ROLE_ARN" ]]; then
        echo "error: IAM role ARN not provided"
        return 1
    fi

    if ! build_and_zip; then
        echo "error with build/zip"
        return 1
    fi
    echo "successful build/zip"

    if aws lambda create-function \
        --function-name "$FUNCTION_NAME" \
        --runtime provided.al2 \
        --handler bootstrap \
        --zip-file fileb://function.zip \
        --role "$IAM_ROLE_ARN"; then
        echo "successfully created lambda"
    else
        return 1
    fi

    cleanup
    return 0
}

update_function() {
    if ! build_and_zip; then
        echo "error with build/zip"
        return 1
    fi
    echo "successful build/zip"

    if aws lambda update-function-code \
        --function-name "$FUNCTION_NAME" \
        --zip-file fileb://function.zip; then
        echo "successfully updated lambda"
    else
        echo "error updating lambda"
        return 1
    fi

    cleanup
    return 0
}

delete_function() {
    if aws lambda delete-function --function-name "$FUNCTION_NAME"; then
        echo "successfully deleted lambda"
    else
        echo "error deleting lambda"
        return 1
    fi

    return 0
}

create_role() {
    ROLE_NAME=$1
    if [[ -z "$ROLE_NAME" ]]; then
        echo "error: role name not provided"
        return 1
    fi

    TRUST_POLICY=$(
        cat <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
    )

    CLOUDWATCH_POLICY=$(
        cat <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:PutRetentionPolicy"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    }
  ]
}
EOF
    )

    if ! aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document "$TRUST_POLICY" >/dev/null; then
        echo "error creating role"
        return 1
    fi
    echo "successfully created role: $ROLE_NAME"

    if ! aws iam put-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-name CloudWatchLogsPolicy \
        --policy-document "$CLOUDWATCH_POLICY" >/dev/null; then
        echo "error attaching policy"
        return 1
    fi
    echo "successfully attached CloudWatch Logs policy"

    ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"
    echo "role ARN: $ROLE_ARN"

    return 0
}

delete_role() {
    ROLE_NAME=$1
    if [[ -z "$ROLE_NAME" ]]; then
        echo "error: role name not provided"
        return 1
    fi

    if ! aws iam delete-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-name CloudWatchLogsPolicy >/dev/null; then
        echo "error deleting inline policy"
        return 1
    fi
    echo "successfully deleted inline policy"

    if ! aws iam delete-role --role-name "$ROLE_NAME" >/dev/null; then
        echo "error deleting role"
        return 1
    fi
    echo "successfully deleted role: $ROLE_NAME"

    return 0
}

full_deploy() {
    FUNCTION_NAME=$1
    ROLE_NAME="$FUNCTION_NAME-role"

    if [[ -z "$FUNCTION_NAME" ]]; then
        echo "function name not provided; using 'cloudwatch-gateway'"
        FUNCTION_NAME="cloudwatch-gateway"
        ROLE_NAME="cloudwatch-gateway-role"
    fi

    if ! create_role "$ROLE_NAME"; then
        echo "error during role creation"
        return 1
    fi

    ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

    echo "waiting for IAM propagation..."
    sleep 10

    if ! create_function "$FUNCTION_NAME" "$ROLE_ARN"; then
        echo "error during function creation"
        return 1
    fi

    return 0
}

full_delete() {
    FUNCTION_NAME=$1
    ROLE_NAME="$FUNCTION_NAME-role"

    if [[ -z "$FUNCTION_NAME" ]]; then
        echo "error: function name not provided"
        return 1
    fi

    if ! delete_function; then
        echo "error during function deletion"
        return 1
    fi

    if ! delete_role "$ROLE_NAME"; then
        echo "error during role deletion"
        return 1
    fi

    return 0
}

usage() {
    cat <<EOD
deploy.sh is a set of commands to create/manage the cloudwatch-gateway lambda.
It assumes you have the AWS CLI (aws) already installed and authenticated on your
device.

Usage:
    deploy.sh [command]

Commands:
    build-and-zip:                          builds and creates a zip of the binary in
                                            the current directory

    full-deploy [function-name]:            creates role, builds, and deploys lambda
                                            if no function name is provided, uses "cloudwatch-gateway"
                                            example: deploy.sh full-deploy cloudwatch-gateway

    full-delete [function-name]:            deletes lambda and role
                                            example: deploy.sh full-delete cloudwatch-gateway

    create-role [role-name]:                creates an IAM role with CloudWatch Logs policy
                                            example: deploy.sh create-role "cloudwatch-gateway-role"

    delete-role [role-name]:                deletes the IAM role
                                            example: deploy.sh delete-role "cloudwatch-gateway-role"

    create [function-name] [role-arn]:      builds and creates the lambda function in AWS
                                            example: deploy.sh create cloudwatch-gateway "arn:aws:iam::123456789:role/RoleName"

    update [function-name]:                 rebuilds and updates the lambda function in AWS
                                            example: deploy.sh update "cloudwatch-gateway"

    delete [function-name]:                 deletes the lambda function from AWS
                                            example: deploy.sh delete "cloudwatch-gateway"
EOD
}

cleanup() {
    rm bootstrap
    rm -r function.zip
}

case "$1" in
"build-zip") build_and_zip ;;
"full-deploy") full_deploy "$2" ;;
"full-delete") full_delete "$2" ;;
"create-role") create_role "$2" ;;
"delete-role") delete_role "$2" ;;
"create") create_function "$2" "$3" ;;
"update") update_function ;;
"delete") delete_function ;;
"help") usage ;;
*)
    echo "invalid argument: $1"
    usage
    ;;
esac
