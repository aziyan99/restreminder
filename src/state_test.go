package main

import (
	"testing"
)

func TestStateMachine_InitialState(t *testing.T) {
	cfg := &Config{
		FocusDuration:        25,
		ShortRestDuration:    5,
		LongRestDuration:     15,
		CyclesBeforeLongRest: 4,
	}
	sm := NewStateMachine(cfg)
	if sm.Mode != ModeIdle {
		t.Errorf("expected initial mode ModeIdle, got %v", sm.Mode)
	}
}

func TestStateMachine_StartStop(t *testing.T) {
	cfg := &Config{
		FocusDuration:        25,
		ShortRestDuration:    5,
		LongRestDuration:     15,
		CyclesBeforeLongRest: 4,
	}
	sm := NewStateMachine(cfg)

	// Test Start
	sm.Start()
	if sm.Mode != ModeFocus {
		t.Errorf("expected mode ModeFocus, got %v", sm.Mode)
	}
	if sm.RemainingSeconds != 25*60 {
		t.Errorf("expected 1500 remaining seconds, got %d", sm.RemainingSeconds)
	}
	if sm.CurrentCycle != 1 {
		t.Errorf("expected current cycle 1, got %d", sm.CurrentCycle)
	}

	// Test Stop
	sm.Stop()
	if sm.Mode != ModeIdle {
		t.Errorf("expected mode ModeIdle after Stop, got %v", sm.Mode)
	}
}

func TestStateMachine_Transitions(t *testing.T) {
	cfg := &Config{
		FocusDuration:        25,
		ShortRestDuration:    5,
		LongRestDuration:     15,
		CyclesBeforeLongRest: 2, // 2 cycles for faster test transitions
	}
	sm := NewStateMachine(cfg)

	var lastOldMode, lastNewMode Mode
	sm.OnModeChange = func(oldMode, newMode Mode) {
		lastOldMode = oldMode
		lastNewMode = newMode
	}

	focusCompleted := false
	sm.OnFocusComplete = func(duration int) {
		focusCompleted = true
		if duration != 25*60 {
			t.Errorf("expected focus complete duration %d, got %d", 25*60, duration)
		}
	}

	sm.Start()
	if lastOldMode != ModeIdle || lastNewMode != ModeFocus {
		t.Errorf("OnModeChange callback not triggered or incorrect: old=%v, new=%v", lastOldMode, lastNewMode)
	}

	// Tick until 0 to complete focus 1
	sm.RemainingSeconds = 1
	sm.Tick()

	if !focusCompleted {
		t.Error("expected OnFocusComplete callback to be triggered")
	}
	if sm.Mode != ModeShortRest {
		t.Errorf("expected transition to ModeShortRest, got %v", sm.Mode)
	}
	if sm.RemainingSeconds != 5*60 {
		t.Errorf("expected short rest duration %d, got %d", 5*60, sm.RemainingSeconds)
	}
	if sm.CurrentCycle != 1 {
		t.Errorf("expected current cycle to remain 1 before rest completion, got %d", sm.CurrentCycle)
	}

	// Complete short rest
	sm.RemainingSeconds = 1
	sm.Tick()
	if sm.Mode != ModeFocus {
		t.Errorf("expected transition back to ModeFocus, got %v", sm.Mode)
	}
	if sm.CurrentCycle != 2 {
		t.Errorf("expected current cycle to increment to 2, got %d", sm.CurrentCycle)
	}
	if sm.RemainingSeconds != 25*60 {
		t.Errorf("expected focus duration 1500, got %d", sm.RemainingSeconds)
	}

	// Tick until 0 to complete focus 2
	focusCompleted = false
	sm.RemainingSeconds = 1
	sm.Tick()
	if !focusCompleted {
		t.Error("expected second focus session completion callback")
	}
	// Since cycles_before_long_rest is 2, it should transition to ModeLongRest
	if sm.Mode != ModeLongRest {
		t.Errorf("expected transition to ModeLongRest, got %v", sm.Mode)
	}
	if sm.RemainingSeconds != 15*60 {
		t.Errorf("expected long rest duration %d, got %d", 15*60, sm.RemainingSeconds)
	}

	// Complete long rest
	sm.RemainingSeconds = 1
	sm.Tick()
	if sm.Mode != ModeIdle {
		t.Errorf("expected transition to ModeIdle, got %v", sm.Mode)
	}
}
