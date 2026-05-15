// Package analyzer provides habit pattern analysis from user actions.
package analyzer

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/habit/pkg/model"
)

// Analyzer analyzes user actions to identify habits and preferences.
type Analyzer struct {
	minOccurrences int     // Minimum times a pattern must appear to become a habit
	confThreshold  float64 // Minimum confidence to consider a habit valid
}

// New creates a new analyzer.
func New(minOccurrences int, confThreshold float64) *Analyzer {
	if minOccurrences <= 0 {
		minOccurrences = model.DefaultMinOccurrences
	}
	if confThreshold <= 0 {
		confThreshold = model.DefaultConfThreshold
	}
	return &Analyzer{
		minOccurrences: minOccurrences,
		confThreshold:  confThreshold,
	}
}

// Analyze analyzes user actions to identify habits.
func (a *Analyzer) Analyze(ctx context.Context, userID string, actions []*model.Action) ([]*model.Habit, error) {
	if len(actions) == 0 {
		return []*model.Habit{}, nil
	}

	var habits []*model.Habit

	// Analyze tool usage patterns
	toolHabits := a.analyzeToolUsage(userID, actions)
	habits = append(habits, toolHabits...)

	// Analyze output style preferences
	styleHabits := a.analyzeOutputStyle(userID, actions)
	habits = append(habits, styleHabits...)

	// Analyze workflow patterns
	workflowHabits := a.analyzeWorkflows(userID, actions)
	habits = append(habits, workflowHabits...)

	// Sort by confidence descending
	sort.Slice(habits, func(i, j int) bool {
		return habits[i].Confidence > habits[j].Confidence
	})

	return habits, nil
}

// GetPreferences extracts preferences from actions.
func (a *Analyzer) GetPreferences(ctx context.Context, userID string, actions []*model.Action) ([]*model.Preference, error) {
	if len(actions) == 0 {
		return []*model.Preference{}, nil
	}

	var prefs []*model.Preference

	// Extract tool usage preferences
	toolPrefs := a.extractToolPrefs(userID, actions)
	prefs = append(prefs, toolPrefs...)

	// Extract style preferences
	stylePrefs := a.extractStylePrefs(userID, actions)
	prefs = append(prefs, stylePrefs...)

	return prefs, nil
}

// analyzeToolUsage identifies frequently used tools as habits.
func (a *Analyzer) analyzeToolUsage(userID string, actions []*model.Action) []*model.Habit {
	toolCounts := make(map[string]int)
	toolContexts := make(map[string]map[string]int)

	for _, action := range actions {
		for _, tool := range action.ToolsUsed {
			toolCounts[tool]++
			if toolContexts[tool] == nil {
				toolContexts[tool] = make(map[string]int)
			}
			// Track context keywords
			if action.Input != "" {
				toolContexts[tool][action.Input]++
			}
		}
	}

	var habits []*model.Habit
	totalActions := len(actions)

	for tool, count := range toolCounts {
		if count < a.minOccurrences {
			continue
		}

		confidence := float64(count) / float64(totalActions)
		if confidence < a.confThreshold {
			continue
		}

		// Find top context keywords
		contexts := toolContexts[tool]
		keywords := topKeys(contexts, model.DefaultTopKeywords)

		habits = append(habits, &model.Habit{
			ID:       uuid.New().String(),
			UserID:   userID,
			Name:     fmt.Sprintf(model.TemplateToolUsageHabit, tool),
			Category: model.CategoryToolUsage,
			Pattern: model.Pattern{
				ToolUsage: []string{tool},
				Keywords:  keywords,
			},
			Frequency: model.Frequency{
				TotalCount: count,
				Trend:      model.TrendStable,
			},
			Confidence: confidence,
			LastSeen:   time.Now(),
		})
	}

	return habits
}

// analyzeOutputStyle identifies output style preferences.
func (a *Analyzer) analyzeOutputStyle(userID string, actions []*model.Action) []*model.Habit {
	styleCounts := make(map[string]int)

	for _, action := range actions {
		if action.OutputStyle != "" {
			styleCounts[action.OutputStyle]++
		}
	}

	var habits []*model.Habit
	totalStyled := 0
	for _, count := range styleCounts {
		totalStyled += count
	}

	if totalStyled < a.minOccurrences {
		return habits
	}

	for style, count := range styleCounts {
		confidence := float64(count) / float64(totalStyled)
		if confidence < a.confThreshold {
			continue
		}

		habits = append(habits, &model.Habit{
			ID:       uuid.New().String(),
			UserID:   userID,
			Name:     fmt.Sprintf(model.TemplateOutputStyleHabit, style),
			Category: model.CategoryStyle,
			Pattern: model.Pattern{
				Context: style,
			},
			Frequency: model.Frequency{
				TotalCount: count,
				Trend:      model.TrendStable,
			},
			Confidence: confidence,
			LastSeen:   time.Now(),
		})
	}

	return habits
}

// analyzeWorkflows identifies common tool sequences as workflow habits.
func (a *Analyzer) analyzeWorkflows(userID string, actions []*model.Action) []*model.Habit {
	// Look for common pairs of tools used together
	pairCounts := make(map[string]int)

	for _, action := range actions {
		tools := action.ToolsUsed
		if len(tools) < 2 {
			continue
		}
		for i := 0; i < len(tools)-1; i++ {
			pair := tools[i] + model.WorkflowPairSep + tools[i+1]
			pairCounts[pair]++
		}
	}

	var habits []*model.Habit
	totalActions := len(actions)

	for pair, count := range pairCounts {
		if count < a.minOccurrences {
			continue
		}

		confidence := float64(count) / float64(totalActions)
		if confidence < a.confThreshold {
			continue
		}

		habits = append(habits, &model.Habit{
			ID:       uuid.New().String(),
			UserID:   userID,
			Name:     fmt.Sprintf(model.TemplateWorkflowHabit, pair),
			Category: model.CategoryWorkflow,
			Pattern: model.Pattern{
				CommandSeq: []string{pair},
			},
			Frequency: model.Frequency{
				TotalCount: count,
				Trend:      model.TrendStable,
			},
			Confidence: confidence,
			LastSeen:   time.Now(),
		})
	}

	return habits
}

// extractToolPrefs extracts tool preferences from actions.
func (a *Analyzer) extractToolPrefs(userID string, actions []*model.Action) []*model.Preference {
	toolCounts := make(map[string]int)
	for _, action := range actions {
		for _, tool := range action.ToolsUsed {
			toolCounts[tool]++
		}
	}

	var prefs []*model.Preference
	for tool, count := range toolCounts {
		if count >= a.minOccurrences {
			prefs = append(prefs, &model.Preference{
				ID:        uuid.New().String(),
				UserID:    userID,
				Category:  model.CategoryToolUsage,
				Name:      fmt.Sprintf("tool_%s", tool),
				Value:     model.PrefValuePreferred,
				Source:    model.SourceImplicit,
				UpdatedAt: time.Now(),
			})
		}
	}

	return prefs
}

// extractStylePrefs extracts style preferences from actions.
func (a *Analyzer) extractStylePrefs(userID string, actions []*model.Action) []*model.Preference {
	styleCounts := make(map[string]int)
	for _, action := range actions {
		if action.OutputStyle != "" {
			styleCounts[action.OutputStyle]++
		}
	}

	var prefs []*model.Preference
	for style, count := range styleCounts {
		if count >= a.minOccurrences {
			prefs = append(prefs, &model.Preference{
				ID:        uuid.New().String(),
				UserID:    userID,
				Category:  model.CategoryStyle,
				Name:      model.PrefOutputStyle,
				Value:     style,
				Source:    model.SourceImplicit,
				UpdatedAt: time.Now(),
			})
		}
	}

	return prefs
}

// topKeys returns the top N keys by value from a map.
func topKeys(m map[string]int, n int) []string {
	type kv struct {
		Key   string
		Value int
	}

	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	var result []string
	for i, kv := range sorted {
		if i >= n {
			break
		}
		result = append(result, kv.Key)
	}
	return result
}
