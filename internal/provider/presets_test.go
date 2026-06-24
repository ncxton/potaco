package provider

import "testing"

func TestGetPresetOpenAI(t *testing.T) {
	p, ok := GetPreset("openai")
	if !ok {
		t.Fatal("preset 'openai' should exist")
	}
	if p.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want https://api.openai.com", p.BaseURL)
	}
	if p.DefaultModel != "dall-e-3" {
		t.Errorf("DefaultModel = %q, want dall-e-3", p.DefaultModel)
	}
	if len(p.Sizes) == 0 {
		t.Error("Sizes should not be empty")
	}
}

func TestGetPresetTogether(t *testing.T) {
	p, ok := GetPreset("together")
	if !ok {
		t.Fatal("preset 'together' should exist")
	}
	if p.BaseURL != "https://api.together.ai" {
		t.Errorf("BaseURL = %q, want https://api.together.ai", p.BaseURL)
	}
}

func TestGetPresetFal(t *testing.T) {
	p, ok := GetPreset("fal")
	if !ok {
		t.Fatal("preset 'fal' should exist")
	}
	if p.BaseURL != "https://fal.run" {
		t.Errorf("BaseURL = %q, want https://fal.run", p.BaseURL)
	}
}

func TestGetPresetUnknown(t *testing.T) {
	_, ok := GetPreset("nonexistent")
	if ok {
		t.Fatal("GetPreset should return false for unknown preset")
	}
}

func TestAllPresets(t *testing.T) {
	presets := AllPresets()
	if len(presets) < 3 {
		t.Errorf("expected at least 3 presets, got %d", len(presets))
	}
	if _, ok := presets["openai"]; !ok {
		t.Error("presets should contain 'openai'")
	}
	if _, ok := presets["together"]; !ok {
		t.Error("presets should contain 'together'")
	}
	if _, ok := presets["fal"]; !ok {
		t.Error("presets should contain 'fal'")
	}
}
