package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
        "TODO API Server. Version 0.0.1 Developed by Nelson Isioma \n")  
        fmt.Fprintf(flag.CommandLine.Output(), "Copyright " + strconv.Itoa(time.Now().Local().Year()) + "\n")
        fmt.Fprintln(flag.CommandLine.Output(), "Usage information:")
        flag.PrintDefaults()
	}

	host := flag.String("h", "localhost", "Server host")
	port := flag.Int("p", 8888, "Server port")
	todoFile := flag.String("f", "todoServer.json", "todo JSON file")
	flag.Parse()

	s := &http.Server{
		Addr: fmt.Sprintf("%s:%d", *host, *port),
		Handler: newMux(*todoFile),
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Local server starting on port %v", *port)
	if err := s.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}