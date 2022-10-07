export OBSERVE_CUSTOMER="$(~/observe/s/testbox get-current-customer)"
export OBSERVE_DOMAIN=observe-sandbox.com:3443
export OBSERVE_USER_EMAIL=sandbox@observeinc.com
export OBSERVE_USER_PASSWORD="very very secret"
export OBSERVE_INSECURE=true
make testacc
