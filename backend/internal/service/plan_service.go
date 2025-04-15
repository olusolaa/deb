package service

import (
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/llm"
	"bibleapp/backend/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// PlanService defines the interface for managing reading plans.
type PlanService interface {
	CreatePlan(ctx context.Context, userID string, topic string, durationDays int, targetAudience string) (domain.ReadingPlan, error)
	GetActiveVerseForToday(ctx context.Context, userID string) (domain.DailyVerse, error)
	ListPlans(ctx context.Context, userID string) ([]domain.ReadingPlan, error)
	// New method to get a verse with its full content fetched on-demand
	GetEnrichedVerseForToday(ctx context.Context, userID string, verseService VerseService) (domain.DailyVerse, error)
}

type planService struct {
	planRepo  repository.PlanRepository
	llmClient llm.LLMClient
	modelName string // Model name from environment config
}

// NewPlanService creates a new PlanService.
func NewPlanService(repo repository.PlanRepository, llmClient llm.LLMClient, modelName string) PlanService {
	return &planService{
		planRepo:  repo,
		llmClient: llmClient,
		modelName: modelName,
	}
}

func (c *planService) generateReadingPlan(ctx context.Context, topic string, durationDays int, targetAudience string) (domain.ReadingPlan, error) {
	var plan domain.ReadingPlan // Return an empty plan on error

	// --- Construct the prompt for plan generation ---
	// This prompt is designed to only get verse references and brief explanations, NOT full verse text
	systemPrompt := fmt.Sprintf(`Create a %d-day Bible reading plan on "%s" for a %s.

For each day, provide ONLY:
1. Day number
2. Bible reference only (no full verse text)
3. A short title (5 words max)

Output ONLY a JSON object with format: {"daily_verses": [{"day": 1, "reference": "John 1:1-5", "text": "", "title": "Short Title Here", "explanation": ""}]}

Make sure to use the "title" field, NOT "explanation" field for the short title.
The actual verse text and explanations will be fetched separately later.
NEVER include actual verse text.`, durationDays, topic, targetAudience)

	userPrompt := fmt.Sprintf(`Create a %d-day reading plan on "%s". Return only the JSON object with verse references.`, durationDays, topic)

	// Use the model from config, not hardcoded
	request := llm.ChatCompletionRequest{
		Model: c.modelName, // Use the model configured in environment
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   800, // Further reduce token consumption to stay within limits
		Temperature: 0.3, // Lower temperature for more focused, structured output
		ResponseFormat: &llm.ResponseFormat{ // Request JSON output if the model/API supports it
			Type: "json_object",
		},
	}

	// Add ResponseFormat struct definition if needed (some APIs use this)
	// type ResponseFormat struct { Type string `json:"type"` }

	llmResponse, err := c.llmClient.CreateChatCompletion(ctx, request)
	if err != nil {
		return plan, fmt.Errorf("LLM completion failed during plan generation: %w", err)
	}

	if len(llmResponse.Choices) == 0 || llmResponse.Choices[0].Message.Content == "" {
		return plan, errors.New("LLM returned an empty response for the plan")
	}

	rawJson := llmResponse.Choices[0].Message.Content
	log.Printf("DEBUG: Raw LLM JSON response for plan:\n%s", rawJson) // Log the raw response for debugging

	// --- Parse the LLM's JSON response ---
	// The LLM should ideally return ONLY the JSON object as requested.
	// Sometimes they add ```json ... ``` markers, try to strip them.
	if strings.HasPrefix(rawJson, "```json") {
		rawJson = strings.TrimPrefix(rawJson, "```json")
		rawJson = strings.TrimSuffix(rawJson, "```")
		rawJson = strings.TrimSpace(rawJson)
	}

	// We expect the JSON structure: {"daily_verses": [...]}
	var planData struct {
		DailyVerses []domain.DailyVerse `json:"daily_verses"`
	}

	err = json.Unmarshal([]byte(rawJson), &planData)
	if err != nil {
		// Log the raw JSON that failed to parse
		log.Printf("ERROR: Failed to unmarshal LLM JSON response into plan structure. Raw JSON: %s", rawJson)
		return plan, fmt.Errorf("failed to parse LLM JSON response for plan: %w", err)
	}

	if len(planData.DailyVerses) == 0 {
		return plan, errors.New("LLM generated a plan with no daily verses")
	}

	// Populate the rest of the plan details
	// Note: UserID is set in the CreatePlan method, not here
	plan.Topic = topic
	plan.DurationDays = durationDays
	plan.TargetAudience = targetAudience
	plan.DailyVerses = planData.DailyVerses
	// ID and CreatedAt will be set by the repository when saving

	return plan, nil

}

func (s *planService) CreatePlan(ctx context.Context, userID string, topic string, durationDays int, targetAudience string) (domain.ReadingPlan, error) {
	if topic == "" || durationDays <= 0 || targetAudience == "" {
		return domain.ReadingPlan{}, errors.New("topic, positive duration, and target audience are required")
	}
	log.Printf("INFO: Requesting LLM to generate plan for topic='%s', duration=%d days, audience='%s'", topic, durationDays, targetAudience)

	// Generate the plan using the LLM
	plan, err := s.generateReadingPlan(ctx, topic, durationDays, targetAudience)
	if err != nil {
		log.Printf("ERROR: Failed to generate plan via LLM: %v", err)
		return domain.ReadingPlan{}, fmt.Errorf("failed to generate reading plan: %w", err)
	}

	// Set the user ID
	plan.UserID = userID

	// Validate basic plan structure
	if len(plan.DailyVerses) == 0 {
		log.Printf("ERROR: LLM generated an empty plan for topic '%s'", topic)
		return domain.ReadingPlan{}, errors.New("LLM generated an empty plan")
	}
	// Ideally, check if len(plan.DailyVerses) roughly matches durationDays, but LLM might adjust.

	// Save the generated plan
	err = s.planRepo.Save(ctx, &plan)
	if err != nil {
		log.Printf("ERROR: Failed to save generated plan: %v", err)
		return domain.ReadingPlan{}, fmt.Errorf("failed to save reading plan: %w", err)
	}

	// The saved plan now has an ID and CreatedAt timestamp
	// Refetch it to return the complete object (optional but good practice)
	savedPlan, err := s.planRepo.FindByID(ctx, plan.ID.String())
	if err != nil || savedPlan == nil {
		log.Printf("WARN: Failed to refetch saved plan %s, returning generated plan: %v", plan.ID, err)
		return plan, nil // Return the original plan if refetch fails or returns nil
	}

	log.Printf("INFO: Successfully created and saved plan %s for topic '%s'", savedPlan.ID, topic)
	return *savedPlan, nil
}

func (s *planService) GetActiveVerseForToday(ctx context.Context, userID string) (domain.DailyVerse, error) {
	// Get the latest plan (considered active)
	// Note: We're still using a placeholder implementation that gets all plans and uses the first one
	plans, err := s.planRepo.FindByUser(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Failed to get plans: %v", err)
		return domain.DailyVerse{}, fmt.Errorf("could not retrieve plans: %w", err)
	}

	if len(plans) == 0 {
		log.Println("INFO: No reading plans found.")
		return domain.DailyVerse{}, fmt.Errorf("no active reading plan found")
	}

	// For simplicity, consider the first plan as the active plan
	activePlan := plans[0]

	// Calculate which day of the plan it is
	// Use UTC to avoid timezone issues with day calculation
	now := time.Now().UTC()
	planStart := activePlan.CreatedAt.UTC()
	// Calculate days passed since the start date (midnight) of the plan creation day
	daysPassed := int(now.Sub(planStart.Truncate(24*time.Hour)).Hours() / 24)

	currentDayNumber := daysPassed + 1 // Day 1 is the day it was created

	log.Printf("INFO: Active plan %s created on %s. Today is day %d of the plan.", activePlan.ID, activePlan.CreatedAt, currentDayNumber)

	if currentDayNumber > activePlan.DurationDays {
		log.Printf("INFO: Active plan %s has finished (duration %d days, current day %d)", activePlan.ID, activePlan.DurationDays, currentDayNumber)
		return domain.DailyVerse{}, repository.ErrDayOutOfRange // Use a specific error maybe "ErrPlanFinished"
	}

	// Get the verse for the calculated day number
	verse, found := activePlan.GetVerseForDay(currentDayNumber)
	if !found {
		// This might happen if LLM didn't generate enough days or data is corrupted
		log.Printf("ERROR: Verse for day %d not found in active plan %s", currentDayNumber, activePlan.ID)
		return domain.DailyVerse{}, fmt.Errorf("verse for day %d not found in the active plan", currentDayNumber)
	}

	return verse, nil
}

func (s *planService) ListPlans(ctx context.Context, userID string) ([]domain.ReadingPlan, error) {
	// Get all plans for this user
	plans, err := s.planRepo.FindByUser(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Failed to list plans: %v", err)
		return nil, fmt.Errorf("failed to retrieve plan list: %w", err)
	}
	// Convert from []*domain.ReadingPlan to []domain.ReadingPlan
	result := make([]domain.ReadingPlan, len(plans))
	for i, plan := range plans {
		result[i] = *plan
	}
	return result, nil
}

// GetEnrichedVerseForToday gets today's verse and enriches it with full text content on-demand
func (s *planService) GetEnrichedVerseForToday(ctx context.Context, userID string, verseService VerseService) (domain.DailyVerse, error) {
	// First, get today's verse from the active plan
	verse, err := s.GetActiveVerseForToday(ctx, userID)
	if err != nil {
		return domain.DailyVerse{}, err
	}

	// Use the verse service to fetch the full content
	enrichedVerse, err := verseService.EnrichDailyVerse(ctx, verse)
	if err != nil {
		log.Printf("ERROR: Failed to enrich verse with content: %v", err)
		return verse, fmt.Errorf("failed to fetch verse content: %w", err)
	}

	return enrichedVerse, nil
}
