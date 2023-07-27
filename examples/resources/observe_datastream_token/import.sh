# Set SECRET to import with the token secret
SECRET=the-token-secret terraform import observe_datastream_token.example 1414010

# Otherwise, because token secrets cannot be read from the API, it will be null
terraform import observe_datastream_token.example 1414010
