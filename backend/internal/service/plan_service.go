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
	CreatePlan(ctx context.Context, topic string, durationDays int, targetAudience string) (domain.ReadingPlan, error)
	GetActiveVerseForToday(ctx context.Context) (domain.DailyVerse, error)
	ListPlans(ctx context.Context) ([]domain.ReadingPlan, error)
}

type planService struct {
	planRepo  repository.PlanRepository
	llmClient llm.LLMClient
}

// NewPlanService creates a new PlanService.
func NewPlanService(repo repository.PlanRepository, llmClient llm.LLMClient) PlanService {
	return &planService{
		planRepo:  repo,
		llmClient: llmClient,
	}
}

func (c *planService) generateReadingPlan(ctx context.Context, topic string, durationDays int, targetAudience string) (domain.ReadingPlan, error) {
	var plan domain.ReadingPlan // Return an empty plan on error

	// --- Construct the complex prompt for plan generation ---
	// This prompt is CRITICAL and may need significant tuning.
	// We explicitly ask for JSON output.
	systemPrompt := fmt.Sprintf(`You are an expert Bible study planner creating a reading plan for a %s.
The plan should focus on the topic: "%s".
The plan duration is %d days.
For EACH day, provide:
1.  A concise Bible reference (e.g., "Genesis 1:1-5" or "Psalm 23:1-3"). Keep daily readings digestible, not too long.
2.  The FULL TEXT of the verses for that reference.
3.  A VERY BRIEF, simple, encouraging explanation (1-2 sentences) suitable for the target audience, explaining the core idea of that day's reading.
4.  Ensure the verses selected are appropriate and understandable for the audience, avoiding overly complex theological debates, obscure laws, excessive violence, or genealogies unless essential to the topic (and if essential, explain simply). Focus on narrative, core teachings, psalms, proverbs, and key events related to the topic.

IMPORTANT: Respond ONLY with a valid JSON object representing the plan. The JSON object should have a single key "daily_verses" which is an array. Each element in the array should be an object with the keys "day" (int, 1-based day number), "reference" (string), "text" (string), and "explanation" (string). Do NOT include any other text, preamble, or explanation outside the JSON structure. Example format for one day: {"day": 1, "reference": "John 1:1-5", "text": "In the beginning was the Word...", "explanation": "This introduces Jesus as the Word..."}
`, targetAudience, topic, durationDays)

	userPrompt := fmt.Sprintf(`Generate the %d-day reading plan about "%s" for a %s in the specified JSON format.`, durationDays, topic, targetAudience)

	request := llm.ChatCompletionRequest{
		Model: "openai/gpt-4o", // Use a powerful model capable of following complex instructions and JSON formatting. GPT-3.5 might struggle. Adjust if needed.
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   1500, // Increase max tokens significantly for plan generation
		Temperature: 0.5,  // Lower temperature for more focused, structured output
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
	plan.Topic = topic
	plan.DurationDays = durationDays
	plan.TargetAudience = targetAudience
	plan.DailyVerses = planData.DailyVerses
	// ID and CreatedAt will be set by the repository when saving

	return plan, nil

}

func (s *planService) CreatePlan(ctx context.Context, topic string, durationDays int, targetAudience string) (domain.ReadingPlan, error) {
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

	// Validate basic plan structure
	if len(plan.DailyVerses) == 0 {
		log.Printf("ERROR: LLM generated an empty plan for topic '%s'", topic)
		return domain.ReadingPlan{}, errors.New("LLM generated an empty plan")
	}
	// Ideally, check if len(plan.DailyVerses) roughly matches durationDays, but LLM might adjust.

	// Save the generated plan
	err = s.planRepo.SavePlan(ctx, &plan)
	if err != nil {
		log.Printf("ERROR: Failed to save generated plan: %v", err)
		return domain.ReadingPlan{}, fmt.Errorf("failed to save reading plan: %w", err)
	}

	// The saved plan now has an ID and CreatedAt timestamp
	// Refetch it to return the complete object (optional but good practice)
	savedPlan, err := s.planRepo.GetPlanByID(ctx, plan.ID)
	if err != nil {
		log.Printf("WARN: Failed to refetch saved plan %s, returning generated plan: %v", plan.ID, err)
		return plan, nil // Return the original plan if refetch fails
	}

	log.Printf("INFO: Successfully created and saved plan %s for topic '%s'", savedPlan.ID, topic)
	return savedPlan, nil
}

func (s *planService) GetActiveVerseForToday(ctx context.Context) (domain.DailyVerse, error) {
	// Get the latest plan (considered active)
	activePlan, err := s.planRepo.GetLatestPlan(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNoActivePlan) {
			log.Println("INFO: No active reading plan found.")
			return domain.DailyVerse{}, err // Propagate specific error
		}
		log.Printf("ERROR: Failed to get latest plan: %v", err)
		return domain.DailyVerse{}, fmt.Errorf("could not retrieve active plan: %w", err)
	}

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

func (s *planService) ListPlans(ctx context.Context) ([]domain.ReadingPlan, error) {
	plans, err := s.planRepo.ListAllPlans(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to list plans: %v", err)
		return nil, fmt.Errorf("failed to retrieve plan list: %w", err)
	}
	return plans, nil
}
