# cloudwatch-gateway

Lambda function that accepts JSON requests and writes structured logs to CloudWatch Logs. Auto-creates log groups and streams as needed.

## Request Format

```json
{
  "log_group": "string (required)",
  "level": "string (optional: debug|info|warn|error, default: info)",
  "message": "string (required)",
  "attributes": [
    {"key": "string", "value": any}
  ],
  "retention_days": 30
}
```

## Response Format

```json
{
  "success": true,
  "message": "Log written successfully"
}
```

Error responses include `"error"` field with details.

## CloudWatch Log Output

Logs are written as JSON with the following structure:

```json
{
  "level": "INFO",
  "message": "your message",
  "custom_attr": "value"
}
```

CloudWatch provides timestamp automatically.

## Deployment

### Prerequisites

- AWS CLI configured and authenticated
- Go installed

## Manual Deployment

If you know what you're doing in AWS and would prefer to do everything manually, you can grab the `function_x86_64` or `function_arm64` zip file from the [latest release](https://github.com/dsrosen6/cloudwatch-gateway/releases/latest).

You can upload that zip file in the Lambda function; it contains the binary `bootstrap` which will be called by the function.

### Quick Deploy

Deploy everything in one command. Clone this repo, and then:

```bash
./deploy.sh full-deploy cloudwatch-gateway
```

Optionally, you can provide a different name as the argument.

This creates the IAM role (provided name with `role` appended), builds the binary, and deploys the Lambda function.

If you would like to perform the tasks modularly, the commands are as follows. You can call the script with `help` to get the same readout:

```txt
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
```

## API Gateway Integration (Optional)

To add authentication and HTTP endpoint access, place the Lambda behind API Gateway as a POST method. Otherwise, just call via the Lambda function URL as a POST.
