package main

import (
	"context"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func TestEndpoint(w http.ResponseWriter, r *http.Request) {
	doHardJob(r.Context())
	if _, err := w.Write([]byte("Hard job done!")); err != nil {
		log.Printf("can't response for TestEndpoint request: %s", err)
		w.WriteHeader(520)
		return
	}
	w.WriteHeader(200)
}
func doHardJob(ctx context.Context) {
	rand.Seed(time.Now().UnixNano())
	timeForHardWork := time.Duration(rand.Intn(30)+1) * time.Second
	select {
	case <-ctx.Done():
		log.Printf("hard job - failed")
	case <-time.After(timeForHardWork):
		log.Printf("hard job - ok")
	}
}
func main() {
	maxGracefulShutdownTime := 15 * time.Second

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	router := mux.NewRouter()
	router.HandleFunc("/test", TestEndpoint).Methods("GET")

	srv := &http.Server{Addr: ":8080",
		Handler:      router,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 6 * time.Second,
		IdleTimeout:  9 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), maxGracefulShutdownTime)
		defer stop()

		//go func() {
		//	<-shutdownCtx.Done()
		//	if shutdownCtx.Err() == context.DeadlineExceeded {
		//		log.Println(fmt.Errorf("shutdown server timeout exceeded: %w", shutdownCtx.Err()))
		//	}
		//}()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %s", err)
		} else {
			log.Print("server exited properly")
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("listen error: %s\n", err)
	}
}
