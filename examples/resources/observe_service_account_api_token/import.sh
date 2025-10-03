# Import format is {service_account_id}:{token_id}
# Import format is {service_account_id}:{token_id}
# Set SECRET to import with the token secret
SECRET=the-token-secret terraform import observe_service_account_token.automation_token 1414000:abcd1234

# Otherwise, because token secrets cannot be read from the API, it will be null
terraform import observe_service_account_token.automation_token 1414000:abcd1234
