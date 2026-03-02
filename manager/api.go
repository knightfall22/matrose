package manager

import "net/http"

type Api struct {
	Address string
	Port    int
	Manager *Manager
	Router  *http.ServeMux
}
