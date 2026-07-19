package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

var version = "0.2.3"

func main() {
	showVersion := flag.Bool("v", false, "show version and exit")
	showVersionLong := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion || *showVersionLong {
		fmt.Println(version)
		os.Exit(0)
	}

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	sm := NewStateMachine(cfg)

	sm.OnFocusComplete = func(durationSeconds int) {
		t := time.Now().UTC()
		historyLog := FocusLog{
			Timestamp: t.Format(time.RFC3339),
			Duration:  durationSeconds,
			Status:    "completed",
		}
		if err := AppendHistory(historyLog); err != nil {
			log.Printf("Error logging history: %v", err)
		}
	}

	fmt.Printf("Starting restreminder desktop application v%s...\n", version)
	RunUI(sm)
}
