package main

type Mode string

const (
	ModeIdle      Mode = "idle"
	ModeFocus     Mode = "focus"
	ModeShortRest Mode = "short_rest"
	ModeLongRest  Mode = "long_rest"
)

type StateMachine struct {
	Mode             Mode
	RemainingSeconds int
	CurrentCycle     int // 1-based, starts at 1 when focus starts
	Config           *Config
	// Callback functions for when a mode starts
	OnModeChange func(oldMode, newMode Mode)
	// Callback function when a focus session completes (for history logging)
	OnFocusComplete func(durationSeconds int)
}

func NewStateMachine(cfg *Config) *StateMachine {
	return &StateMachine{
		Mode:             ModeIdle,
		RemainingSeconds: 0,
		CurrentCycle:     1,
		Config:           cfg,
	}
}

func (sm *StateMachine) Start() {
	if sm.Mode != ModeIdle {
		return
	}
	oldMode := sm.Mode
	sm.Mode = ModeFocus
	sm.RemainingSeconds = sm.Config.FocusDuration * 60
	sm.CurrentCycle = 1
	if sm.OnModeChange != nil {
		sm.OnModeChange(oldMode, sm.Mode)
	}
}

func (sm *StateMachine) StartShortRest() {
	oldMode := sm.Mode
	sm.Mode = ModeShortRest
	sm.RemainingSeconds = sm.Config.ShortRestDuration * 60
	if sm.OnModeChange != nil {
		sm.OnModeChange(oldMode, sm.Mode)
	}
}

func (sm *StateMachine) StartLongRest() {
	oldMode := sm.Mode
	sm.Mode = ModeLongRest
	sm.RemainingSeconds = sm.Config.LongRestDuration * 60
	if sm.OnModeChange != nil {
		sm.OnModeChange(oldMode, sm.Mode)
	}
}

func (sm *StateMachine) Stop() {
	if sm.Mode == ModeIdle {
		return
	}
	oldMode := sm.Mode
	sm.Mode = ModeIdle
	sm.RemainingSeconds = 0
	sm.CurrentCycle = 1
	if sm.OnModeChange != nil {
		sm.OnModeChange(oldMode, sm.Mode)
	}
}

func (sm *StateMachine) Tick() {
	if sm.Mode == ModeIdle {
		return
	}

	if sm.RemainingSeconds > 0 {
		sm.RemainingSeconds--
		if sm.RemainingSeconds > 0 {
			return
		}
	}

	// remaining seconds reached 0, transition!
	oldMode := sm.Mode
	switch oldMode {
	case ModeFocus:
		// Focus completed! Log history
		if sm.OnFocusComplete != nil {
			sm.OnFocusComplete(sm.Config.FocusDuration * 60)
		}

		if sm.CurrentCycle >= sm.Config.CyclesBeforeLongRest {
			sm.Mode = ModeLongRest
			sm.RemainingSeconds = sm.Config.LongRestDuration * 60
		} else {
			sm.Mode = ModeShortRest
			sm.RemainingSeconds = sm.Config.ShortRestDuration * 60
		}
	case ModeShortRest:
		// Short rest completed, increment cycle and back to focus
		sm.CurrentCycle++
		sm.Mode = ModeFocus
		sm.RemainingSeconds = sm.Config.FocusDuration * 60
	case ModeLongRest:
		// Long rest completed, go back to idle
		sm.Mode = ModeIdle
		sm.RemainingSeconds = 0
		sm.CurrentCycle = 1
	}

	if sm.OnModeChange != nil {
		sm.OnModeChange(oldMode, sm.Mode)
	}
}
