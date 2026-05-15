package tools

import (
	"context"
	"fmt"

	auratools "github.com/oneliang/aura/tools/pkg"
)

// WeatherTool demonstrates custom tool implementation.
type WeatherTool struct{}

// NewWeatherTool creates a new weather tool.
func NewWeatherTool() *WeatherTool {
	return &WeatherTool{}
}

// Name returns the tool name.
func (t *WeatherTool) Name() string {
	return "weather"
}

// Description returns the tool description.
func (t *WeatherTool) Description() string {
	return "Get weather information for a location. Usage: weather(location)"
}

// Execute runs the tool with given parameters.
func (t *WeatherTool) Execute(ctx context.Context, params map[string]any) (*auratools.ToolResult, error) {
	location, ok := params["location"].(string)
	if !ok {
		return &auratools.ToolResult{
			Status: auratools.ToolStatusError,
			Error:  "location parameter required (string)",
		}, nil
	}

	// Simulated weather response
	result := fmt.Sprintf("Weather in %s: Sunny, 25°C", location)
	return &auratools.ToolResult{
		Status:  auratools.ToolStatusSuccess,
		Content: result,
		Data: map[string]any{
			"location":    location,
			"condition":   "sunny",
			"temperature": 25,
		},
	}, nil
}