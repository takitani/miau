package services

import (
	"testing"
	"time"
)

func TestParseTimeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHour int
		wantMin  int
	}{
		{
			name:     "valid morning time",
			input:    "09:00",
			wantHour: 9,
			wantMin:  0,
		},
		{
			name:     "valid afternoon time",
			input:    "14:30",
			wantHour: 14,
			wantMin:  30,
		},
		{
			name:     "midnight",
			input:    "00:00",
			wantHour: 0,
			wantMin:  0,
		},
		{
			name:     "end of day",
			input:    "23:59",
			wantHour: 23,
			wantMin:  59,
		},
		{
			name:     "invalid - too short",
			input:    "9:00",
			wantHour: 9, // defaults to 9
			wantMin:  0,
		},
		{
			name:     "empty string",
			input:    "",
			wantHour: 9, // default
			wantMin:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHour, gotMin := parseTimeString(tt.input)
			if gotHour != tt.wantHour {
				t.Errorf("parseTimeString(%q) hour = %d, want %d", tt.input, gotHour, tt.wantHour)
			}
			if gotMin != tt.wantMin {
				t.Errorf("parseTimeString(%q) min = %d, want %d", tt.input, gotMin, tt.wantMin)
			}
		})
	}
}

func TestSchedulePresets(t *testing.T) {
	// Create service with default settings
	svc := &ScheduledSendService{
		defaultMorning:   "09:00",
		defaultAfternoon: "14:00",
	}

	presets := svc.GetPresetTimes()

	// Verify presets are in the future
	now := time.Now()

	if presets.TomorrowMorning.Before(now) {
		t.Error("TomorrowMorning should be in the future")
	}

	if presets.TomorrowAfternoon.Before(now) {
		t.Error("TomorrowAfternoon should be in the future")
	}

	if presets.MondayMorning.Before(now) {
		t.Error("MondayMorning should be in the future")
	}

	// Verify morning is before afternoon for same day
	if presets.TomorrowMorning.After(presets.TomorrowAfternoon) {
		t.Error("TomorrowMorning should be before TomorrowAfternoon")
	}

	// Verify Monday is a Monday
	if presets.MondayMorning.Weekday() != time.Monday {
		t.Errorf("MondayMorning weekday = %v, want Monday", presets.MondayMorning.Weekday())
	}

	// Verify times match config
	if presets.MorningTime != "09:00" {
		t.Errorf("MorningTime = %q, want %q", presets.MorningTime, "09:00")
	}
	if presets.AfternoonTime != "14:00" {
		t.Errorf("AfternoonTime = %q, want %q", presets.AfternoonTime, "14:00")
	}
}

func TestScheduledSendServiceStartStop(t *testing.T) {
	// Create service without dependencies (they won't be used in this test)
	svc := &ScheduledSendService{
		stopChan:      make(chan struct{}),
		checkInterval: 100 * time.Millisecond,
	}

	// Initially not running
	if svc.IsRunning() {
		t.Error("Service should not be running initially")
	}

	// Start the service
	svc.Start()
	time.Sleep(10 * time.Millisecond) // Give time to start

	if !svc.IsRunning() {
		t.Error("Service should be running after Start()")
	}

	// Stop the service
	svc.Stop()
	time.Sleep(10 * time.Millisecond) // Give time to stop

	if svc.IsRunning() {
		t.Error("Service should not be running after Stop()")
	}
}

func TestScheduledSendServiceSetters(t *testing.T) {
	svc := &ScheduledSendService{
		stopChan:      make(chan struct{}),
		checkInterval: time.Minute,
	}

	// Test SetCheckInterval
	svc.SetCheckInterval(5 * time.Minute)
	if svc.checkInterval != 5*time.Minute {
		t.Errorf("checkInterval = %v, want %v", svc.checkInterval, 5*time.Minute)
	}

	// Test SetPresets
	svc.SetPresets("08:00", "15:00")
	if svc.defaultMorning != "08:00" {
		t.Errorf("defaultMorning = %q, want %q", svc.defaultMorning, "08:00")
	}
	if svc.defaultAfternoon != "15:00" {
		t.Errorf("defaultAfternoon = %q, want %q", svc.defaultAfternoon, "15:00")
	}

	// Test SetNotifications
	svc.SetNotifications(false, true)
	if svc.notifyOnSend != false {
		t.Error("notifyOnSend should be false")
	}
	if svc.notifyOnFail != true {
		t.Error("notifyOnFail should be true")
	}
}
