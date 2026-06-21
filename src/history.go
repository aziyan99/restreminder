package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type FocusLog struct {
	Timestamp string `json:"timestamp"`
	Duration  int    `json:"duration"`
	Status    string `json:"status"`
}

func GetHistoryFilePath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

func LoadHistory() ([]FocusLog, error) {
	filePath, err := GetHistoryFilePath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []FocusLog{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var logs []FocusLog
	if err := json.Unmarshal(data, &logs); err != nil {
		// If the file is corrupted, return empty list to avoid crash
		return []FocusLog{}, nil
	}

	return logs, nil
}

func SaveHistory(logs []FocusLog) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dir, "history.json")
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func AppendHistory(log FocusLog) error {
	logs, err := LoadHistory()
	if err != nil {
		logs = []FocusLog{}
	}
	logs = append(logs, log)
	return SaveHistory(logs)
}
