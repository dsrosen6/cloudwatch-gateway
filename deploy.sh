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
    IAM_ROLE_ARN=$1
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
        --function-name cloudwatch-gateway \
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

cleanup() {
    rm bootstrap
    rm -r function.zip
}

case "$1" in
"create") create_function "$2" ;;
"update") update_function ;;
"delete") delete_function ;;
*) echo "invalid argument: $1" ;;
esac
