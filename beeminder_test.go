package main

import (
	"testing"
)

// TestCreateGoalValidation tests basic validation for CreateGoal function
func TestCreateGoalValidation(t *testing.T) {
	// This is a unit test that verifies the CreateGoal function signature
	// We can't actually call the API without valid credentials, but we can
	// verify the function exists and has the correct signature

	config := &Config{
		Username:  "testuser",
		AuthToken: "testtoken",
	}

	// Test that the function can be called (will fail due to invalid credentials)
	// but this ensures the function signature is correct
	_, err := CreateGoal(config, "testgoal", "Test Goal", "hustler", "units", "null", "100", "1")

	// We expect an error since we don't have valid credentials
	// But this test ensures the function compiles and can be called
	if err == nil {
		t.Log("Unexpected success with test credentials")
	}
}

// TestGoalCreatedMsgStructure tests that goalCreatedMsg exists
func TestGoalCreatedMsgStructure(t *testing.T) {
	msg := goalCreatedMsg{
		goal: &Goal{Slug: "test"},
		err:  nil,
	}

	if msg.goal.Slug != "test" {
		t.Errorf("Expected goal slug to be 'test', got %s", msg.goal.Slug)
	}
}
