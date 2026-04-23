package traps

import (
	"context"
	"fmt"
	"net/http"
)

type APITrap struct {
	Server *http.Server
	Port   int
}

func NewAPITrap(port int) *APITrap {
	mux := http.NewServeMux()
	
	// Create a trigger channel
	trigger := make(chan bool, 1)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If any request hits this server, the trap is sprung
		trigger <- true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "success", "internal_token": "trap-secret-12345"}`)
	})

	return &APITrap{
		Port: port,
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}

func (a *APITrap) Start(ctx context.Context, trigger chan bool) {
	go func() {
		// Forward inner trigger to outer
		// This is a bit simplified; in a real system we'd use a better signaling mechanism
		fmt.Printf("API Trap listening on %d...\n", a.Port)
		if err := a.Server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("API Trap server error: %v\n", err)
		}
	}()
}

func (a *APITrap) Stop() {
	a.Server.Close()
}
