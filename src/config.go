package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultFocusDuration        = 25
	DefaultShortRestDuration    = 5
	DefaultLongRestDuration     = 15
	DefaultCyclesBeforeLongRest = 4
)

type Config struct {
	FocusDuration        int `json:"focus_duration"`
	ShortRestDuration    int `json:"short_rest_duration"`
	LongRestDuration     int `json:"long_rest_duration"`
	CyclesBeforeLongRest int `json:"cycles_before_long_rest"`
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "restreminder"), nil
}

func (c *Config) Validate() error {
	if c.FocusDuration < 1 || c.FocusDuration > 180 {
		return fmt.Errorf("focus_duration must be between 1 and 180")
	}
	if c.ShortRestDuration < 1 || c.ShortRestDuration > 60 {
		return fmt.Errorf("short_rest_duration must be between 1 and 60")
	}
	if c.LongRestDuration < 1 || c.LongRestDuration > 120 {
		return fmt.Errorf("long_rest_duration must be between 1 and 120")
	}
	if c.CyclesBeforeLongRest < 1 || c.CyclesBeforeLongRest > 12 {
		return fmt.Errorf("cycles_before_long_rest must be between 1 and 12")
	}
	return nil
}

func LoadConfig() (*Config, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(dir, "settings.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		cfg := &Config{
			FocusDuration:        DefaultFocusDuration,
			ShortRestDuration:    DefaultShortRestDuration,
			LongRestDuration:     DefaultLongRestDuration,
			CyclesBeforeLongRest: DefaultCyclesBeforeLongRest,
		}
		if err := SaveConfig(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		cfg = Config{
			FocusDuration:        DefaultFocusDuration,
			ShortRestDuration:    DefaultShortRestDuration,
			LongRestDuration:     DefaultLongRestDuration,
			CyclesBeforeLongRest: DefaultCyclesBeforeLongRest,
		}
		_ = SaveConfig(&cfg)
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dir, "settings.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
