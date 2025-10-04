# Import a service account token using the composite ID format: <service_account_id>/<token_id>
# The service account ID is the numeric ID from the service account
# The token ID is the unique identifier for the token
terraform import observe_service_account_token.example 12345/token-abc-123

# Set SECRET environment variable to import with the token secret
# Otherwise, because token secrets cannot be read from the API, it will be null
SECRET=the-token-secret terraform import observe_service_account_token.example 12345/token-abc-123

