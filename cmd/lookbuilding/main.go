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

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	l.Logger.Infof(`listening on %s`, addr)

	go worker.Worker()

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
