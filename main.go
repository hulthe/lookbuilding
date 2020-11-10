package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	Logger logrus.Logger = *logrus.New()
)

func main() {
	addr, isPresent := os.LookupEnv(ENV_ADDR)
	if !isPresent {
		addr = "0.0.0.0:8000"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		TriggerScan()
		fmt.Fprintf(w, "OK")
	})

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	Logger.Infof(`listening on %s`, addr)

	go Worker()

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
