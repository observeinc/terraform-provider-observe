package api

// GQLClient runs GQL queries
type GQLClient interface {
	Run(request string, vars map[string]interface{}) (map[string]interface{}, error)
}

// Client implements our current API given an interface that can speak GraphQL
type Client struct {
	GQLClient
}

// New returns client to native API
func New(c GQLClient) *Client {
	return &Client{c}
}
