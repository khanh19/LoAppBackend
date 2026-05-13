// Service hello implements simple starter endpoints.
package hello

import "context"

//encore:api public path=/hello/:name
func World(ctx context.Context, name string) (*Response, error) {
	return &Response{Message: "Hello, " + name + "!"}, nil
}
