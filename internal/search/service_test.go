package search

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSearchServiceImpl_SearchProfiles(t *testing.T) {
	// Create a mock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	// Create the service with mock DB
	service := &SearchServiceImpl{
		db: db,
	}

	t.Run("Basic name search", func(t *testing.T) {
		// Setup expected query and response
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT p.profile_id\\)").
			WithArgs("%John%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery("SELECT (.+) FROM profiles p").
			WithArgs("%John%").
			WillReturnRows(sqlmock.NewRows([]string{
				"profile_id", "user_id", "full_name", "city", "gender", "age", "activity_type", "description",
			}).AddRow(
				1, 101, "John Doe", "New York", "male", 30, "improv", "Improv enthusiast",
			))

		// Set up mock for getImprovProfileData
		mock.ExpectQuery("SELECT goal, looking_for_team FROM improv_profiles").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"goal", "looking_for_team"}).
				AddRow("Career", true))

		mock.ExpectQuery("SELECT style FROM improv_profile_styles").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"style"}).
				AddRow("Short Form").
				AddRow("Long Form"))

		// Execute the search
		req := ProfileSearchRequest{
			FullName: "John",
			Limit:    10,
		}

		resp, err := service.SearchProfiles(context.Background(), req)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Validate the response
		if len(resp.Results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(resp.Results))
		}
		result := resp.Results[0]
		if result.FullName != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%s'", result.FullName)
		}
		if result.City != "New York" {
			t.Errorf("Expected city 'New York', got '%s'", result.City)
		}
		if result.ActivityType != "improv" {
			t.Errorf("Expected activity_type 'improv', got '%s'", result.ActivityType)
		}
		if result.ImprovGoal != "Career" {
			t.Errorf("Expected improv_goal 'Career', got '%s'", result.ImprovGoal)
		}
		if len(result.ImprovStyles) != 2 {
			t.Errorf("Expected 2 improv styles, got %d", len(result.ImprovStyles))
		}
		if *result.ImprovLookingForTeam != true {
			t.Errorf("Expected looking_for_team true, got %v", *result.ImprovLookingForTeam)
		}
	})

	// ...existing code...
	t.Run("Complex filter music profile search", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup complex filters
		ageMin := 25
		ageMax := 40

		// Setup expected query and response for count
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT p.profile_id\\)").
			// Fix parameter order to match actual service implementation
			WithArgs("%rock%", ageMin, ageMax, "music", "guitar").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Setup expected query and response for search
		mock.ExpectQuery("SELECT (.+) FROM profiles p").
			// Fix parameter order to match actual service implementation
			WithArgs("%rock%", ageMin, ageMax, "music", "guitar").
			WillReturnRows(sqlmock.NewRows([]string{
				"profile_id", "user_id", "full_name", "city", "gender", "age", "activity_type", "description",
			}).AddRow(
				2, 102, "Rock Star", "Los Angeles", "male", 35, "music", "Rock musician",
			))

		// Setup mock for getMusicProfileData
		mock.ExpectQuery("SELECT genre_code FROM music_profile_genres").
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"genre_code"}).
				AddRow("rock").
				AddRow("pop"))

		mock.ExpectQuery("SELECT instrument_code FROM music_profile_instruments").
			WithArgs(2).
			WillReturnRows(sqlmock.NewRows([]string{"instrument_code"}).
				AddRow("guitar").
				AddRow("drums"))

		// Execute the search
		req := ProfileSearchRequest{
			FullName:         "rock",
			AgeMin:           &ageMin,
			AgeMax:           &ageMax,
			ActivityType:     "music",
			MusicInstruments: []string{"guitar"},
			Limit:            20,
		}

		resp, err := service.SearchProfiles(context.Background(), req)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Validate the response
		if len(resp.Results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(resp.Results))
		}
		result := resp.Results[0]
		if result.FullName != "Rock Star" {
			t.Errorf("Expected name 'Rock Star', got '%s'", result.FullName)
		}
		if result.ActivityType != "music" {
			t.Errorf("Expected activity_type 'music', got '%s'", result.ActivityType)
		}
		if len(result.MusicGenres) != 2 {
			t.Errorf("Expected 2 music genres, got %d", len(result.MusicGenres))
		}
		if len(result.MusicInstruments) != 2 {
			t.Errorf("Expected 2 music instruments, got %d", len(result.MusicInstruments))
		}
	})
	// ...existing code...

	t.Run("Complex filter improv profile search", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup improv-specific filters
		improvLookingForTeam := true
		improvGoal := "Career"
		improvStyles := []string{"Short Form", "Long Form"}

		// Set up placeholders for IN clause
		styleParams := make([]driver.Value, 0, len(improvStyles))
		for _, style := range improvStyles {
			styleParams = append(styleParams, style)
		}

		// Setup expected query and response for count
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT p.profile_id\\)").
			// Fix parameter order to match actual implementation
			WithArgs("%Sarah%", "improv", styleParams[0], styleParams[1], improvGoal, improvLookingForTeam).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Setup expected query and response for search
		mock.ExpectQuery("SELECT (.+) FROM profiles p").
			// Fix parameter order to match actual implementation
			WithArgs("%Sarah%", "improv", styleParams[0], styleParams[1], improvGoal, improvLookingForTeam).
			WillReturnRows(sqlmock.NewRows([]string{
				"profile_id", "user_id", "full_name", "city", "gender", "age", "activity_type", "description",
			}).AddRow(
				3, 103, "Sarah Smith", "Chicago", "female", 32, "improv", "Experienced improv actor",
			))

		// Setup mock for getImprovProfileData
		mock.ExpectQuery("SELECT goal, looking_for_team FROM improv_profiles").
			WithArgs(3).
			WillReturnRows(sqlmock.NewRows([]string{"goal", "looking_for_team"}).
				AddRow(improvGoal, improvLookingForTeam))

		mock.ExpectQuery("SELECT style FROM improv_profile_styles").
			WithArgs(3).
			WillReturnRows(sqlmock.NewRows([]string{"style"}).
				AddRow("Short Form").
				AddRow("Long Form"))

		// Execute the search
		req := ProfileSearchRequest{
			FullName:             "Sarah",
			ActivityType:         "improv",
			ImprovGoal:           improvGoal,
			ImprovLookingForTeam: &improvLookingForTeam,
			ImprovStyles:         improvStyles,
			Limit:                20,
		}

		resp, err := service.SearchProfiles(context.Background(), req)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Validate the response
		if len(resp.Results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(resp.Results))
		}

		result := resp.Results[0]
		if result.FullName != "Sarah Smith" {
			t.Errorf("Expected name 'Sarah Smith', got '%s'", result.FullName)
		}
		if result.ActivityType != "improv" {
			t.Errorf("Expected activity_type 'improv', got '%s'", result.ActivityType)
		}
		if result.ImprovGoal != improvGoal {
			t.Errorf("Expected improv_goal '%s', got '%s'", improvGoal, result.ImprovGoal)
		}
		if result.ImprovLookingForTeam == nil || *result.ImprovLookingForTeam != improvLookingForTeam {
			t.Errorf("Expected improv_looking_for_team %v, got %v", improvLookingForTeam, result.ImprovLookingForTeam)
		}
		if len(result.ImprovStyles) != 2 {
			t.Errorf("Expected 2 improv styles, got %d", len(result.ImprovStyles))
		}

		// Verify we got the right styles back
		expectedStyles := map[string]bool{
			"Short Form": true,
			"Long Form":  true,
		}

		for _, style := range result.ImprovStyles {
			if !expectedStyles[style] {
				t.Errorf("Unexpected style returned: %s", style)
			}
			// Mark as found
			delete(expectedStyles, style)
		}

		if len(expectedStyles) > 0 {
			t.Errorf("Missing expected styles: %v", expectedStyles)
		}
	})

	t.Run("Count query error", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup error for count query
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT p.profile_id\\)").
			WithArgs("%error%"). // Updated to match actual parameter format
			WillReturnError(errors.New("database error"))

		// Execute the search
		req := ProfileSearchRequest{
			FullName: "error",
			Limit:    10,
		}

		_, err := service.SearchProfiles(context.Background(), req)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		if err.Error() != "failed to count profiles: database error" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Main query error", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup successful count but error for main query
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT p.profile_id\\)").
			WithArgs("%error%"). // Updated to match actual parameter format
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery("SELECT (.+) FROM profiles p").
			WithArgs("%error%"). // Updated to match actual parameter format
			WillReturnError(errors.New("database error"))

		// Execute the search
		req := ProfileSearchRequest{
			FullName: "error",
			Limit:    10,
		}

		_, err := service.SearchProfiles(context.Background(), req)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		if err.Error() != "failed to search profiles: database error" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Row scan error", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup successful count but error when scanning rows
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT p.profile_id\\)").
			WithArgs("%error%"). // Updated to match actual parameter format
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Return rows with wrong column count to cause scan error
		mock.ExpectQuery("SELECT (.+) FROM profiles p").
			WithArgs("%error%"). // Updated to match actual parameter format
			WillReturnRows(sqlmock.NewRows([]string{"profile_id"}).AddRow(1))

		// Execute the search
		req := ProfileSearchRequest{
			FullName: "error",
			Limit:    10,
		}

		_, err := service.SearchProfiles(context.Background(), req)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		if err.Error() == "" {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestGetImprovProfileData(t *testing.T) {
	// Create a mock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	// Create the service with mock DB
	service := &SearchServiceImpl{
		db: db,
	}

	t.Run("Success case", func(t *testing.T) {
		// Setup mock
		profileID := 1
		mock.ExpectQuery("SELECT goal, looking_for_team FROM improv_profiles").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"goal", "looking_for_team"}).
				AddRow("Career", true))

		mock.ExpectQuery("SELECT style FROM improv_profile_styles").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"style"}).
				AddRow("Short Form").
				AddRow("Long Form"))

		// Call the function
		result, err := service.getImprovProfileData(context.Background(), profileID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Validate result
		if result.Goal != "Career" {
			t.Errorf("Expected goal 'Career', got '%s'", result.Goal)
		}
		if *result.LookingForTeam != true {
			t.Errorf("Expected looking_for_team true, got %v", *result.LookingForTeam)
		}
		if len(result.Styles) != 2 {
			t.Errorf("Expected 2 styles, got %d", len(result.Styles))
		}
	})

	t.Run("No rows found", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup mock for no rows
		profileID := 2
		mock.ExpectQuery("SELECT goal, looking_for_team FROM improv_profiles").
			WithArgs(profileID).
			WillReturnError(sql.ErrNoRows)

		// Call the function
		result, err := service.getImprovProfileData(context.Background(), profileID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Validate empty result
		if result.Goal != "" {
			t.Errorf("Expected empty goal, got '%s'", result.Goal)
		}
		if result.LookingForTeam != nil {
			t.Errorf("Expected nil looking_for_team, got %v", result.LookingForTeam)
		}
		if len(result.Styles) != 0 {
			t.Errorf("Expected 0 styles, got %d", len(result.Styles))
		}
	})

	t.Run("Database error", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup mock for database error
		profileID := 3
		mock.ExpectQuery("SELECT goal, looking_for_team FROM improv_profiles").
			WithArgs(profileID).
			WillReturnError(errors.New("database error"))

		// Call the function
		_, err := service.getImprovProfileData(context.Background(), profileID)
		if err == nil {
			t.Fatal("Expected an error but got none")
		}
		if err.Error() != "database error" {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestGetMusicProfileData(t *testing.T) {
	// Create a mock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	// Create the service with mock DB
	service := &SearchServiceImpl{
		db: db,
	}

	t.Run("Success case", func(t *testing.T) {
		// Setup mock
		profileID := 1
		mock.ExpectQuery("SELECT genre_code FROM music_profile_genres").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"genre_code"}).
				AddRow("rock").
				AddRow("jazz"))

		mock.ExpectQuery("SELECT instrument_code FROM music_profile_instruments").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"instrument_code"}).
				AddRow("guitar").
				AddRow("piano"))

		// Call the function
		result, err := service.getMusicProfileData(context.Background(), profileID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Validate result
		if len(result.Genres) != 2 {
			t.Errorf("Expected 2 genres, got %d", len(result.Genres))
		}
		if result.Genres[0] != "rock" || result.Genres[1] != "jazz" {
			t.Errorf("Expected genres rock and jazz, got %v", result.Genres)
		}
		if len(result.Instruments) != 2 {
			t.Errorf("Expected 2 instruments, got %d", len(result.Instruments))
		}
		if result.Instruments[0] != "guitar" || result.Instruments[1] != "piano" {
			t.Errorf("Expected instruments guitar and piano, got %v", result.Instruments)
		}
	})

	t.Run("Empty results", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup mock for empty results
		profileID := 2
		mock.ExpectQuery("SELECT genre_code FROM music_profile_genres").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"genre_code"}))

		mock.ExpectQuery("SELECT instrument_code FROM music_profile_instruments").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"instrument_code"}))

		// Call the function
		result, err := service.getMusicProfileData(context.Background(), profileID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Validate empty result
		if len(result.Genres) != 0 {
			t.Errorf("Expected 0 genres, got %d", len(result.Genres))
		}
		if len(result.Instruments) != 0 {
			t.Errorf("Expected 0 instruments, got %d", len(result.Instruments))
		}
	})

	t.Run("Database error", func(t *testing.T) {
		// Reset expectations
		mock.ExpectationsWereMet()

		// Setup mock for database error
		profileID := 3
		mock.ExpectQuery("SELECT genre_code FROM music_profile_genres").
			WithArgs(profileID).
			WillReturnError(errors.New("database error"))

		// Call the function
		_, err := service.getMusicProfileData(context.Background(), profileID)
		if err == nil {
			t.Fatal("Expected an error but got none")
		}
		if err.Error() != "database error" {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
