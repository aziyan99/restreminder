package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
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

	fmt.Println("Starting restreminder desktop application...")
	RunUI(sm)
}
