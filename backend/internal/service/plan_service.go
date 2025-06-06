package service

import (
	"bibleapp/backend/internal/config"
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/llm"
	"bibleapp/backend/internal/repository"
	"bibleapp/backend/internal/util" // Added for IsValidReference
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
	// Auto-generate default weekly plan based on the yearly theme
	EnsureDefaultWeeklyPlan(ctx context.Context, yearlyTheme string, targetAudience string) error
	// Get a verse for a specific date
	GetVerseForDate(ctx context.Context, userID string, date time.Time) (domain.DailyVerse, error)
	// Get an enriched verse for a specific date
	GetEnrichedVerseForDate(ctx context.Context, userID string, date time.Time, verseService VerseService) (domain.DailyVerse, error)
	// Delete a plan by ID
	DeletePlan(ctx context.Context, planID string, userID string) error
	// Update a plan
	UpdatePlan(ctx context.Context, plan domain.ReadingPlan, userID string) error
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
2. Bible reference (format specified below)
3. A short title (5 words max)

IMPORTANT: Bible reference formats:
- Single verse: "BookName Chapter:Verse" (e.g., "John 3:16")
- Verse range: "BookName Chapter:StartVerse-EndVerse" (e.g., "John 3:16-18")
- Multi-chapter: "BookName StartChapter:StartVerse-EndChapter:EndVerse" (e.g., "Matthew 5:1-7:29")
- Whole chapter: Include verses (e.g., "John 3:1-36" not just "John 3")
- Multiple books in same day: Use comma-separated format (e.g., "John 21:25, Acts 1:1-5")

Use full standard book names with proper spacing (e.g., "1 John 1:1" not "1John 1:1" or "First John 1:1").

Output ONLY JSON: {"daily_verses": [{"day": 1, "reference": "John 1:1-5, Psalms 1:1-6", "text": "", "title": "Beginning and Blessing", "explanation": ""}]}

Use "title" field for short title. NEVER include verse text.`, durationDays, topic, targetAudience)

	originalUserPrompt := fmt.Sprintf(`Create a %d-day reading plan on "%s". Return only the JSON object with verse references.`, durationDays, topic)
	userPrompt := originalUserPrompt

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

	const maxRetries = 3
	var lastError error

	// --- Retry Loop for LLM Call, Parsing, and Validation ---
	for retry := 0; retry < maxRetries; retry++ {

		// Update messages for retries
		if retry > 0 {
			// Replace last user message with the updated prompt containing feedback
			if len(request.Messages) > 0 && request.Messages[len(request.Messages)-1].Role == "user" {
				request.Messages[len(request.Messages)-1].Content = userPrompt
			} else {
				// Should not happen with our structure, but fallback just in case
				request.Messages = append(request.Messages, llm.Message{Role: "user", Content: userPrompt})
			}
			log.Printf("INFO: Retrying plan generation (attempt %d/%d) for topic '%s' due to validation errors.", retry+1, maxRetries, topic)
		}

		llmResponse, err := c.llmClient.CreateChatCompletion(ctx, request)
		if err != nil {
			lastError = fmt.Errorf("LLM completion failed (attempt %d/%d): %w", retry+1, maxRetries, err)
			// Don't retry on API errors, return directly
			log.Printf("ERROR: %s", lastError.Error())
			return plan, lastError
		}

		if len(llmResponse.Choices) == 0 || llmResponse.Choices[0].Message.Content == "" {
			lastError = fmt.Errorf("LLM returned empty response (attempt %d/%d)", retry+1, maxRetries)
			userPrompt = originalUserPrompt + "\n\nThe previous attempt returned an empty response. Please provide the JSON plan."
			continue // Retry
		}

		rawJson := llmResponse.Choices[0].Message.Content
		log.Printf("DEBUG: Raw LLM JSON response (attempt %d/%d):\n%s", retry+1, maxRetries, rawJson)

		// --- Parse the LLM's JSON response ---
		if strings.HasPrefix(rawJson, "```json") {
			rawJson = strings.TrimPrefix(rawJson, "```json")
			rawJson = strings.TrimSuffix(rawJson, "```")
			rawJson = strings.TrimSpace(rawJson)
		}

		var planData struct {
			DailyVerses []domain.DailyVerse `json:"daily_verses"`
		}

		err = json.Unmarshal([]byte(rawJson), &planData)
		if err != nil {
			lastError = fmt.Errorf("failed to parse LLM JSON (attempt %d/%d): %w. Raw JSON: %s", retry+1, maxRetries, err, rawJson)
			log.Printf("ERROR: %s", lastError.Error())
			userPrompt = originalUserPrompt + "\n\nThe previous response was not valid JSON. Please ensure you ONLY return the JSON object, without any surrounding text or markers."
			continue // Retry
		}

		if len(planData.DailyVerses) == 0 {
			lastError = fmt.Errorf("LLM generated plan with no daily verses (attempt %d/%d)", retry+1, maxRetries)
			log.Printf("WARN: %s", lastError.Error())
			userPrompt = originalUserPrompt + "\n\nThe previous response had an empty 'daily_verses' array. Please ensure the array contains the daily plan entries."
			continue // Retry
		}

		// --- === VALIDATION STEP === ---
		invalidRefsWithErrors := make(map[string]string)
		for i, dailyVerse := range planData.DailyVerses {
			if strings.TrimSpace(dailyVerse.Reference) == "" {
				invalidRefsWithErrors[fmt.Sprintf("Day %d", i+1)] = "reference field is empty"
				continue
			}
			// Handle comma-separated references within a single day's entry
			individualRefs := strings.Split(dailyVerse.Reference, ",")
			for _, singleRef := range individualRefs {
				trimmedRef := strings.TrimSpace(singleRef)
				if trimmedRef == "" {
					continue
				} // Skip empty strings resulting from split

				isValid, validationErr := util.IsValidReference(trimmedRef)
				if !isValid {
					invalidRefsWithErrors[trimmedRef] = validationErr.Error()
				}
			}
		}

		// --- === CHECK VALIDATION RESULTS === ---
		if len(invalidRefsWithErrors) == 0 {
			// Success! Populate the plan and return
			log.Printf("INFO: Successfully generated and validated reading plan for '%s' after %d attempt(s).", topic, retry+1)
			plan.Topic = topic
			plan.DurationDays = durationDays
			plan.TargetAudience = targetAudience
			plan.DailyVerses = planData.DailyVerses
			return plan, nil // <<< SUCCESS EXIT
		}

		// --- Validation Failed - Prepare for Retry ---
		lastError = fmt.Errorf("validation failed (attempt %d/%d): %d invalid references found", retry+1, maxRetries, len(invalidRefsWithErrors))
		log.Printf("WARN: %s. Invalid references: %v", lastError.Error(), invalidRefsWithErrors)

		// Construct feedback prompt
		feedback := "\n\nThe previous plan contained invalid or incorrectly formatted references. Please correct the following:\n"
		for ref, reason := range invalidRefsWithErrors {
			feedback += fmt.Sprintf("- '%s': %s\n", ref, reason)
		}
		feedback += "Ensure all references strictly follow the required formats ('Book Ch:V' or 'Book Ch:V-V') and are complete."
		userPrompt = originalUserPrompt + feedback // Append feedback to original request

		// Loop continues for the next retry
	}

	// If loop finishes, all retries failed
	log.Printf("ERROR: Failed to generate a valid reading plan for topic '%s' after %d retries. Last error: %v", topic, maxRetries, lastError)
	return plan, fmt.Errorf("failed to generate a valid reading plan after %d retries: %w", maxRetries, lastError)
}

func (s *planService) CreatePlan(ctx context.Context, userID string, topic string, durationDays int, targetAudience string) (domain.ReadingPlan, error) {
	if topic == "" || durationDays <= 0 || targetAudience == "" {
		return domain.ReadingPlan{}, errors.New("topic, positive duration, and target audience are required")
	}

	// Check for existing user plans that might overlap with this new plan
	// Only perform this check for regular users, not for the default plan
	if userID != "default" {
		// Calculate the date range for the new plan
		starting := time.Now().Truncate(24 * time.Hour)  // Start today at midnight
		ending := starting.AddDate(0, 0, durationDays-1) // End date is start + (duration-1) days

		// Get existing user plans
		existingPlans, err := s.planRepo.FindByUser(ctx, userID)
		if err != nil {
			log.Printf("ERROR: Failed to check existing user plans: %v", err)
			return domain.ReadingPlan{}, fmt.Errorf("failed to check existing plans: %w", err)
		}

		// Check for date overlap with any existing plan
		for _, existingPlan := range existingPlans {
			// Check if the new plan's date range overlaps with an existing plan
			// Two date ranges overlap if the start of one is before or equal to the end of the other,
			// and the end of one is after or equal to the start of the other
			if (starting.Before(existingPlan.EndDate) || starting.Equal(existingPlan.EndDate)) &&
				(ending.After(existingPlan.StartDate) || ending.Equal(existingPlan.StartDate)) {
				// Found an overlapping plan
				log.Printf("WARN: User %s already has a plan '%s' (ID: %s) overlapping with the requested date range",
					userID, existingPlan.Topic, existingPlan.ID)
				return domain.ReadingPlan{}, fmt.Errorf("you already have a reading plan for this date range (topic: %s)",
					existingPlan.Topic)
			}
		}
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

	// Set calendar dates - ensure they're properly initialized
	now := time.Now().Truncate(24 * time.Hour) // Start today at midnight
	plan.StartDate = now
	plan.EndDate = now.AddDate(0, 0, durationDays-1) // End date is start + (duration-1) days

	// Verify dates are set (debug only)
	log.Printf("DEBUG: Setting plan date range: %s to %s (duration: %d days)",
		plan.StartDate.Format("2006-01-02"), plan.EndDate.Format("2006-01-02"), durationDays)

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
	// Use the current date
	return s.GetVerseForDate(ctx, userID, time.Now())
}

func (s *planService) GetVerseForDate(ctx context.Context, userID string, date time.Time) (domain.DailyVerse, error) {
	// Normalize the date to midnight for consistent comparison
	targetDate := date.Truncate(24 * time.Hour)

	// First, try to find a user-specific plan that covers this date
	plans, err := s.planRepo.FindByUser(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Failed to get user plans: %v", err)
		return domain.DailyVerse{}, fmt.Errorf("could not retrieve plans: %w", err)
	}

	var activePlan *domain.ReadingPlan

	// Find a user plan that covers the target date
	for _, plan := range plans {
		// Log plan dates for debugging
		log.Printf("DEBUG: Checking user plan %s with date range %s to %s against target date %s",
			plan.ID, plan.StartDate.Format("2006-01-02"), plan.EndDate.Format("2006-01-02"), targetDate.Format("2006-01-02"))

		// Check if plan covers target date (note: using not-before/not-after logic to be more inclusive)
		if !targetDate.Before(plan.StartDate) && !targetDate.After(plan.EndDate) {
			activePlan = plan
			log.Printf("INFO: Found user-specific plan %s for user %s covering date %s",
				activePlan.ID, userID, targetDate.Format("2006-01-02"))
			break
		}
	}

	// If no user plan found for this date, try default plans
	if activePlan == nil {
		defaultPlans, err := s.planRepo.FindByUser(ctx, "default")
		if err != nil {
			log.Printf("ERROR: Failed to get default plans: %v", err)
			return domain.DailyVerse{}, fmt.Errorf("could not retrieve default plans: %w", err)
		}

		// Debug info about default plans
		log.Printf("DEBUG: Found %d default plans. Target date: %s", len(defaultPlans), targetDate.Format("2006-01-02"))
		for i, plan := range defaultPlans {
			log.Printf("DEBUG: Default plan #%d: ID %s, Topic '%s', StartDate %s, EndDate %s",
				i+1, plan.ID, plan.Topic, plan.StartDate.Format("2006-01-02"), plan.EndDate.Format("2006-01-02"))
		}

		if len(defaultPlans) == 0 {
			log.Println("INFO: No default or user reading plans found for this date.")
			return domain.DailyVerse{}, fmt.Errorf("no reading plan found for date %s", targetDate.Format("2006-01-02"))
		}

		// ALWAYS use the most recent default plan regardless of date ranges
		// This ensures users always get a reading, even when date ranges don't match
		activePlan = defaultPlans[0] // Always use the most recent default plan (they're sorted by creation date desc)
		log.Printf("INFO: Using most recent default plan %s (topic: %s) for date %s",
			activePlan.ID, activePlan.Topic, targetDate.Format("2006-01-02"))
	}

	if activePlan == nil {
		return domain.DailyVerse{}, fmt.Errorf("no suitable reading plan found for date %s",
			targetDate.Format("2006-01-02"))
	}

	// Calculate which day of the plan it is
	// If the date is before the plan starts, use day 1
	// If the date is after the plan ends, use the last day
	var dayNumber int

	if targetDate.Before(activePlan.StartDate) {
		dayNumber = 1 // Use first day of plan
	} else if targetDate.After(activePlan.EndDate) {
		dayNumber = activePlan.DurationDays // Use last day of plan
	} else {
		// Calculate days since start of plan (add 1 because day 1 is the start date)
		dayNumber = int(targetDate.Sub(activePlan.StartDate).Hours()/24) + 1
	}

	log.Printf("INFO: For date %s, using day %d of plan %s (range %s to %s)",
		targetDate.Format("2006-01-02"),
		dayNumber,
		activePlan.ID,
		activePlan.StartDate.Format("2006-01-02"),
		activePlan.EndDate.Format("2006-01-02"))

	// Get the verse for the calculated day number
	verse, found := activePlan.GetVerseForDay(dayNumber)
	if !found {
		log.Printf("ERROR: Day %d not found in plan %s (has %d verses)",
			dayNumber, activePlan.ID, len(activePlan.DailyVerses))
		return domain.DailyVerse{}, repository.ErrDayOutOfRange
	}

	log.Printf("INFO: Found verse for day %d, reference %s (%s)", dayNumber, verse.Reference, verse.Title)
	return verse, nil
}

func (s *planService) ListPlans(ctx context.Context, userID string) ([]domain.ReadingPlan, error) {
	// Get only user-specific plans
	userPlans, err := s.planRepo.FindByUser(ctx, userID)
	if err != nil {
		log.Printf("ERROR: Failed to list user plans: %v", err)
		return nil, fmt.Errorf("failed to retrieve user plan list: %w", err)
	}

	// Convert from []*domain.ReadingPlan to []domain.ReadingPlan
	result := make([]domain.ReadingPlan, len(userPlans))
	for i, plan := range userPlans {
		result[i] = *plan
	}
	return result, nil
}

// GetEnrichedVerseForToday gets today's verse and enriches it with full text content on-demand
func (s *planService) GetEnrichedVerseForToday(ctx context.Context, userID string, verseService VerseService) (domain.DailyVerse, error) {
	// Use the current date
	return s.GetEnrichedVerseForDate(ctx, userID, time.Now(), verseService)
}

// GetEnrichedVerseForDate gets a verse for a specific date and enriches it with full text content
func (s *planService) GetEnrichedVerseForDate(ctx context.Context, userID string, date time.Time, verseService VerseService) (domain.DailyVerse, error) {
	// First, get the verse for the specified date
	verse, err := s.GetVerseForDate(ctx, userID, date)
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

// StartWeeklyPlanScheduler starts a background process that periodically checks
// if a new default plan needs to be generated, using the yearly theme from config
func StartWeeklyPlanScheduler(planService PlanService, cfg *config.Config) {
	// Run immediately to ensure we have a plan
	go func() {
		// Get yearly theme and target audience from config
		yearlyTheme := cfg.YearlyTheme
		targetAudience := cfg.DefaultTargetAudience

		log.Printf("INFO: Using yearly theme: %s and target audience: %s", yearlyTheme, targetAudience)

		ctx := context.Background()
		err := planService.EnsureDefaultWeeklyPlan(ctx, yearlyTheme, targetAudience)
		if err != nil {
			log.Printf("ERROR: Failed initial default plan generation: %v", err)
		}
	}()

	// Schedule a weekly check on Sundays
	go func() {
		for {
			// Calculate time until next Sunday
			now := time.Now()
			currentDay := int(now.Weekday())
			daysUntilSunday := 0

			// If today is not Sunday (0), calculate days until next Sunday
			if currentDay != 0 {
				daysUntilSunday = 7 - currentDay
			} else {
				// If today is Sunday but we've already passed our usual check time,
				// wait until next Sunday
				if now.Hour() >= 2 { // Assuming we want to run at 2 AM
					daysUntilSunday = 7
				}
			}

			// Calculate next Sunday at 2 AM
			nextSunday := time.Date(
				now.Year(), now.Month(), now.Day()+daysUntilSunday,
				2, 0, 0, 0, now.Location(),
			)

			// Calculate duration to sleep
			timeUntilNextSunday := nextSunday.Sub(now)
			log.Printf("INFO: Next default plan check scheduled for %s (in %s)",
				nextSunday.Format(time.RFC3339), timeUntilNextSunday)

			// Sleep until next Sunday
			time.Sleep(timeUntilNextSunday)

			// Use yearly theme and target audience from config
			yearlyTheme := cfg.YearlyTheme
			targetAudience := cfg.DefaultTargetAudience

			// Generate plan on Sunday
			log.Printf("INFO: Sunday check - generating default plan if needed")
			ctx := context.Background()
			err := planService.EnsureDefaultWeeklyPlan(ctx, yearlyTheme, targetAudience)
			if err != nil {
				log.Printf("ERROR: Failed scheduled default plan generation: %v", err)
			}
		}
	}()

	log.Printf("INFO: Sunday default plan generation scheduler started")
}

// EnsureDefaultWeeklyPlan checks if a current default plan exists and is valid for the current date
// If no valid plan exists, it generates a new 7-day default plan based on the yearly theme
// It ensures topics don't repeat by tracking previous topics
func (s *planService) EnsureDefaultWeeklyPlan(ctx context.Context, yearlyTheme string, targetAudience string) error {
	// Get existing default plans
	defaultPlans, err := s.planRepo.FindByUser(ctx, "default")
	if err != nil {
		return fmt.Errorf("failed to check existing default plans: %w", err)
	}

	// Check if we need to generate a new plan
	needNewPlan := true
	var previousTopics []string

	if len(defaultPlans) > 0 {
		// Check if most recent plan is valid for today's date
		latestPlan := defaultPlans[0] // Plans are already sorted by CreateAt desc
		today := time.Now().Truncate(24 * time.Hour)

		log.Printf("DEBUG: Checking if existing default plan covers today - plan dates: %s to %s, today: %s",
			latestPlan.StartDate.Format("2006-01-02"), latestPlan.EndDate.Format("2006-01-02"), today.Format("2006-01-02"))

		// Use not-before/not-after logic for more reliable date comparison
		if !today.Before(latestPlan.StartDate) && !today.After(latestPlan.EndDate) {
			// Today is within the plan's date range, no need for a new one
			log.Printf("INFO: Current default plan %s is valid for today (date range %s to %s)",
				latestPlan.ID,
				latestPlan.StartDate.Format("2006-01-02"),
				latestPlan.EndDate.Format("2006-01-02"))
			needNewPlan = false
		}

		// Collect all previous topics to avoid repetition
		for _, plan := range defaultPlans {
			previousTopics = append(previousTopics, plan.Topic)
		}
	}

	// If no need for a new plan, exit early
	if !needNewPlan {
		return nil
	}

	// Generate a new default plan
	log.Printf("INFO: Generating new default weekly plan based on yearly theme: %s", yearlyTheme)

	// Generate a topic related to the yearly theme that hasn't been used before
	topic, err := s.generateNewTopicFromTheme(ctx, yearlyTheme, previousTopics)
	if err != nil {
		return fmt.Errorf("failed to generate new topic: %w", err)
	}

	// Create a 7-day plan with the generated topic
	plan, err := s.generateReadingPlan(ctx, topic, 7, targetAudience)
	if err != nil {
		return fmt.Errorf("failed to generate default reading plan: %w", err)
	}

	// Set it as a default plan - ensure the exact string "default" is used
	plan.UserID = "default"

	// Set calendar dates for the plan
	now := time.Now().Truncate(24 * time.Hour) // Start today at midnight
	plan.StartDate = now
	plan.EndDate = now.AddDate(0, 0, 6) // 7-day plan (0-6)

	// Verify dates are set (debug only)
	log.Printf("DEBUG: New default plan date range: %s to %s",
		plan.StartDate.Format("2006-01-02"), plan.EndDate.Format("2006-01-02"))

	// Log key details before saving
	log.Printf("DEBUG: Saving new default plan - UserID: '%s', ID: %s, Topic: '%s'",
		plan.UserID, plan.ID, plan.Topic)

	// Save the plan
	err = s.planRepo.Save(ctx, &plan)
	if err != nil {
		return fmt.Errorf("failed to save default plan: %w", err)
	}

	// Verify the plan was saved correctly by retrieving it
	defaultPlans, verifyErr := s.planRepo.FindByUser(ctx, "default")
	if verifyErr != nil {
		log.Printf("WARN: Could not verify default plan after saving: %v", verifyErr)
	} else if len(defaultPlans) == 0 {
		log.Printf("ERROR: Default plan not found after saving! Verification failed.")
	} else {
		log.Printf("INFO: Verified default plan was saved correctly. Found %d default plans.", len(defaultPlans))
	}

	log.Printf("INFO: Successfully created new default plan with topic '%s' and ID %s", topic, plan.ID)
	return nil
}

// generateNewTopicFromTheme uses the LLM to generate a new topic based on the yearly theme
// It avoids topics that have been used before
func (s *planService) generateNewTopicFromTheme(ctx context.Context, yearlyTheme string, previousTopics []string) (string, error) {
	// Convert the previous topics to a string for the prompt
	previousTopicsStr := strings.Join(previousTopics, ", ")

	// Create prompt for the LLM
	systemPrompt := fmt.Sprintf(`You are helping to generate a topic for a 7-day Bible reading plan.

The yearly theme is: "%s"

Previously used topics: %s

Please suggest a specific, focused topic related to the yearly theme that hasn't been used before. It can be studying a specific book of the Bible or a character in the Bible or a story from the Bible etc.
Your response should be ONLY the topic name, nothing else. Keep it concise (3-7 words).`,
		yearlyTheme, previousTopicsStr)

	userPrompt := "Generate a new Bible reading plan topic based on the yearly theme."

	// Create the LLM request
	request := llm.ChatCompletionRequest{
		Model: s.modelName,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	// Get response from LLM
	llmResponse, err := s.llmClient.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("LLM completion failed during topic generation: %w", err)
	}

	if len(llmResponse.Choices) == 0 || llmResponse.Choices[0].Message.Content == "" {
		return "", errors.New("LLM returned an empty response for topic generation")
	}

	// Extract just the topic text
	topic := strings.TrimSpace(llmResponse.Choices[0].Message.Content)

	// Remove any quotes or backticks
	topic = strings.Trim(topic, "`\"'")

	return topic, nil
}

// DeletePlan deletes a plan by ID, ensures the user has permission
func (s *planService) DeletePlan(ctx context.Context, planID string, userID string) error {
	// First check if the plan exists and belongs to this user
	plan, err := s.planRepo.FindByID(ctx, planID)
	if err != nil {
		return fmt.Errorf("error finding plan before deletion: %w", err)
	}

	if plan == nil {
		return errors.New("plan not found")
	}

	// Verify ownership or admin status
	if plan.UserID != userID && userID != "admin" {
		return errors.New("unauthorized: cannot delete another user's plan")
	}

	// Delete the plan
	return s.planRepo.Delete(ctx, planID)
}

// UpdatePlan updates an existing plan, ensures the user has permission
func (s *planService) UpdatePlan(ctx context.Context, plan domain.ReadingPlan, userID string) error {
	// First check if the plan exists and belongs to this user
	existingPlan, err := s.planRepo.FindByID(ctx, plan.ID.String())
	if err != nil {
		return fmt.Errorf("error finding plan before update: %w", err)
	}

	if existingPlan == nil {
		return errors.New("plan not found")
	}

	// Verify ownership or admin status
	if existingPlan.UserID != userID && userID != "admin" {
		return errors.New("unauthorized: cannot update another user's plan")
	}

	// Update the plan (keep original user ID and creation date)
	plan.UserID = existingPlan.UserID
	plan.CreatedAt = existingPlan.CreatedAt

	return s.planRepo.Save(ctx, &plan)
}
