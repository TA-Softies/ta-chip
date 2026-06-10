package submit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type InspectionData struct {
	PCLocation     string `json:"pc_location"`
	Rounder        string `json:"rounder"`
	ShiftTime      string `json:"shift_time"`
	Display        string `json:"display"`
	MouseKeyboard  string `json:"mouse_keyboard"`
	KensingtonLock string `json:"kensington_lock"`
	Conduiting     string `json:"conduiting"`
	Tidiness       string `json:"tidiness"`
	BootToWindows  string `json:"boot_to_windows"`
	TimeDate       string `json:"time_date"`
	Wallpaper      string `json:"wallpaper"`
	Domain         string `json:"domain"`
	MSOffice       string `json:"microsoft_office"`
	MSTeams        string `json:"microsoft_teams"`
	Browser        string `json:"browser"`
	DFFrozen       string `json:"deepfreeze_frozen"`
	DFPolicy       string `json:"deepfreeze_policy"`
	Remarks        string `json:"remarks"`
	Timestamp      string `json:"timestamp"`
}

type Response struct {
	Success bool   `json:"success"`
	Row     int    `json:"row"`
	Error   string `json:"error"`
}

func Submit(url string, data InspectionData) (*Response, error) {
	if url == "" {
		return nil, fmt.Errorf("appscript_url is not configured")
	}

	data.Timestamp = time.Now().Format("2006-01-02 15:04:05")

	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var result Response
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("response parse error (status %d): %w", resp.StatusCode, err)
	}
	if !result.Success {
		return &result, fmt.Errorf("AppScript error: %s", result.Error)
	}
	return &result, nil
}
