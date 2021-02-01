package main

import (
	"fmt"
	"net/http"
	"os"

	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/worker"
)

const EnvAddr = "LOOKBUILDING_ADDR"

func main() {
	//l.Logger.Level = logrus.DebugLevel

	addr, isPresent := os.LookupEnv(EnvAddr)
	if !isPresent {
		addr = "0.0.0.0:8000"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		worker.TriggerScan()
		fmt.Fprintf(w, "OK")
	})

	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		// TODO: if the last scan errored, this should not return OK
		fmt.Fprintf(w, "OK")
	})

	l.Logger.Infof(`listening on %s`, addr)

	go worker.Worker()

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
