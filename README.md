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

- AWS CLI configured with credentials
- Go 1.x installed locally

### Quick Deploy

Deploy everything in one command:

```bash
./deploy.sh full-deploy cloudwatch-gateway
```

This creates the IAM role (named `cloudwatch-gateway-role`), builds the binary, and deploys the Lambda function.

Delete everything:

```bash
./deploy.sh full-delete cloudwatch-gateway
```

This deletes the Lambda function and its associated IAM role.

### Manual Deploy

#### Create IAM Role

Create a role with inline CloudWatch Logs policy:

```bash
./deploy.sh create-role cloudwatch-gateway-role
```

This outputs the role ARN needed for Lambda creation. The role includes permissions for:

- `logs:CreateLogGroup`
- `logs:CreateLogStream`
- `logs:PutLogEvents`
- `logs:PutRetentionPolicy`

Delete the role:

```bash
./deploy.sh delete-role cloudwatch-gateway-role
```

Alternatively, use an existing IAM role ARN with the required CloudWatch Logs permissions.

#### Deploy Lambda

Create the function:

```bash
./deploy.sh create cloudwatch-gateway arn:aws:iam::123456789012:role/YourLambdaRole
```

Update existing function:

```bash
./deploy.sh update cloudwatch-gateway
```

Delete function:

```bash
./deploy.sh delete cloudwatch-gateway
```

### Test Invocation

## Build Only

To build without deploying:

```bash
./deploy.sh build-zip
```

This creates `function.zip` in the current directory.

```bash
aws lambda invoke \
  --function-name cloudwatch-gateway \
  --payload '{"log_group":"/test/logs","level":"info","message":"test message","attributes":[{"key":"user_id","value":"123"}]}' \
  response.json
```

## API Gateway Integration (Optional)

To add authentication and HTTP endpoint access, place the Lambda behind API Gateway as a POST method. Otherwise, just call via the Lambda function URL as a POST.
