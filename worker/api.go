package worker

import "net/http"

// Provide a means by which the manager communates with a worker.
// Passes instructions through the use of the API
type Api struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *http.ServeMux
}
