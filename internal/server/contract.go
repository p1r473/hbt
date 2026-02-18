package server

import "context"

// Graph has all the functions a suggestion graph needs to be implemented.
type Graph interface {
	// Track adds to the graph the command cmd performed at path wd by the id
	// user/process.
	Track(ctx context.Context, id int, wd, cmd string) error
	// Hint returns the next suggestion for user/process id at path wd.
	Hint(ctx context.Context, id int, wd string) (string, error)
	// End clears a session for user/process id. This is useful to reset a
	// stateful graph.
	End(ctx context.Context, id int) error
	// Delete removes a previously tracked command. It should not return an
	// error.
	Delete(ctx context.Context, id int, wd, cmd string) error
	// Save serialises the graph to the given file path.
	Save(ctx context.Context, filePath string) error
	// Load initialises the graph with a serialiastion at the give file path.
	Load(ctx context.Context, filePath string) error
}
