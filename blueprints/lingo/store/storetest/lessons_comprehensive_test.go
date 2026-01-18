package storetest

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/go-mizu/mizu/blueprints/lingo/store/sqlite"
	"github.com/stretchr/testify/require"
)

// TestLessonDataCompleteness verifies that seed data has complete lessons and exercises
func TestLessonDataCompleteness(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(courses), 1, "Should have at least 1 course")

	for _, course := range courses {
		t.Run(course.Title, func(t *testing.T) {
			// Test course has units
			units, err := s.Courses().GetCoursePath(ctx, course.ID)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(units), 3, "Course should have at least 3 units")

			for _, unit := range units {
				t.Run(unit.Title, func(t *testing.T) {
					// Test unit has skills
					require.GreaterOrEqual(t, len(unit.Skills), 2,
						"Unit %s should have at least 2 skills", unit.Title)

					for _, skill := range unit.Skills {
						t.Run(skill.Name, func(t *testing.T) {
							// Test skill has lessons
							skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
							require.NoError(t, err)
							require.Equal(t, 5, len(skillDetail.Lessons),
								"Skill %s should have exactly 5 lessons (one per level)", skill.Name)

							for _, lesson := range skillDetail.Lessons {
								t.Run("Level"+string(rune('0'+lesson.Level)), func(t *testing.T) {
									// Test lesson has exercises
									exercises, err := s.Courses().GetExercises(ctx, lesson.ID)
									require.NoError(t, err)
									require.GreaterOrEqual(t, len(exercises), 10,
										"Lesson level %d of skill %s should have at least 10 exercises",
										lesson.Level, skill.Name)

									// Validate exercise fields
									for _, ex := range exercises {
										require.NotEmpty(t, ex.Type, "Exercise should have type")
										require.NotEmpty(t, ex.Prompt, "Exercise should have prompt")
										require.NotEmpty(t, ex.CorrectAnswer, "Exercise should have correct answer")
									}
								})
							}
						})
					}
				})
			}
		})
	}
}

// TestExerciseTypeVariety ensures each lesson has multiple exercise types
func TestExerciseTypeVariety(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		for _, unit := range units {
			for _, skill := range unit.Skills {
				skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
				require.NoError(t, err)

				for _, lesson := range skillDetail.Lessons {
					exercises, err := s.Courses().GetExercises(ctx, lesson.ID)
					require.NoError(t, err)

					// Count exercise types
					typeCount := make(map[string]int)
					for _, ex := range exercises {
						typeCount[ex.Type]++
					}

					// Ensure variety (at least 2 different types per lesson)
					require.GreaterOrEqual(t, len(typeCount), 2,
						"Lesson %s level %d should have at least 2 different exercise types, got %d",
						skill.Name, lesson.Level, len(typeCount))
				}
			}
		}
	}
}

// TestRequiredExerciseTypes checks that all required exercise types exist
func TestRequiredExerciseTypes(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	requiredTypes := []string{
		"translation",
		"multiple_choice",
		"word_bank",
		"listening",
		"fill_blank",
		"match_pairs",
	}

	foundTypes := make(map[string]bool)

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		for _, unit := range units {
			for _, skill := range unit.Skills {
				skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
				require.NoError(t, err)

				for _, lesson := range skillDetail.Lessons {
					exercises, err := s.Courses().GetExercises(ctx, lesson.ID)
					require.NoError(t, err)

					for _, ex := range exercises {
						foundTypes[ex.Type] = true
					}
				}
			}
		}
	}

	for _, reqType := range requiredTypes {
		require.True(t, foundTypes[reqType],
			"Exercise type '%s' should exist in seed data", reqType)
	}
}

// TestExerciseChoices verifies multiple choice exercises have proper choices
func TestExerciseChoices(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		for _, unit := range units {
			for _, skill := range unit.Skills {
				skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
				require.NoError(t, err)

				for _, lesson := range skillDetail.Lessons {
					exercises, err := s.Courses().GetExercises(ctx, lesson.ID)
					require.NoError(t, err)

					for _, ex := range exercises {
						switch ex.Type {
						case "multiple_choice":
							// Multiple choice must have choices with correct answer included
							require.GreaterOrEqual(t, len(ex.Choices), 3,
								"Exercise type %s should have at least 3 choices", ex.Type)

							// Verify correct answer is in choices
							found := false
							for _, c := range ex.Choices {
								if c == ex.CorrectAnswer {
									found = true
									break
								}
							}
							require.True(t, found,
								"Correct answer '%s' should be in choices for exercise %s",
								ex.CorrectAnswer, ex.ID)

						case "listening":
							// Listening exercises should have audio URL or be free-form typing
							require.NotEmpty(t, ex.Prompt, "Listening exercise should have a prompt")

						case "translation":
							// Translation exercises are free-form typing, no choices required
							require.NotEmpty(t, ex.CorrectAnswer, "Translation exercise should have correct answer")

						case "word_bank", "fill_blank":
							// Word bank and fill_blank may have choices
							require.NotEmpty(t, ex.CorrectAnswer, "Exercise should have correct answer")

						case "match_pairs":
							// Match pairs should have choices (the pairs)
							require.NotEmpty(t, ex.CorrectAnswer, "Match pairs exercise should have correct answer")
						}
					}
				}
			}
		}
	}
}

// TestSkillLevels ensures skills have correct level structure
func TestSkillLevels(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		for _, unit := range units {
			for _, skill := range unit.Skills {
				skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
				require.NoError(t, err)

				// Check skill has 5 levels
				require.Equal(t, 5, skill.Levels,
					"Skill %s should have 5 levels defined", skill.Name)

				// Check lessons exist for each level
				lessonLevels := make(map[int]bool)
				for _, lesson := range skillDetail.Lessons {
					lessonLevels[lesson.Level] = true
				}

				for level := 1; level <= 5; level++ {
					require.True(t, lessonLevels[level],
						"Skill %s should have a lesson for level %d", skill.Name, level)
				}
			}
		}
	}
}

// TestUnitPositions verifies units are properly ordered
func TestUnitPositions(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		// Verify units are in order
		for i, unit := range units {
			require.Equal(t, i+1, unit.Position,
				"Unit %s should have position %d, got %d", unit.Title, i+1, unit.Position)
		}
	}
}

// TestSkillPositions verifies skills within units are properly ordered
func TestSkillPositions(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		for _, unit := range units {
			for i, skill := range unit.Skills {
				require.Equal(t, i+1, skill.Position,
					"Skill %s should have position %d, got %d", skill.Name, i+1, skill.Position)
			}
		}
	}
}

// TestExerciseDifficulty verifies exercises have valid difficulty values
func TestExerciseDifficulty(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		for _, unit := range units {
			for _, skill := range unit.Skills {
				skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
				require.NoError(t, err)

				for _, lesson := range skillDetail.Lessons {
					exercises, err := s.Courses().GetExercises(ctx, lesson.ID)
					require.NoError(t, err)

					for _, ex := range exercises {
						require.GreaterOrEqual(t, ex.Difficulty, 1,
							"Exercise difficulty should be at least 1")
						require.LessOrEqual(t, ex.Difficulty, 5,
							"Exercise difficulty should be at most 5")
					}
				}
			}
		}
	}
}

// TestCourseLanguages verifies courses have valid language references
func TestCourseLanguages(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	languages, err := s.Courses().ListLanguages(ctx)
	require.NoError(t, err)

	langMap := make(map[string]bool)
	for _, lang := range languages {
		langMap[lang.ID] = true
	}

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		require.True(t, langMap[course.FromLanguageID],
			"Course %s has invalid from_language_id: %s", course.Title, course.FromLanguageID)
		require.True(t, langMap[course.LearningLanguageID],
			"Course %s has invalid learning_language_id: %s", course.Title, course.LearningLanguageID)
	}
}

// TestExerciseCount verifies each lesson has the expected number of exercises
func TestExerciseCount(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		for _, unit := range units {
			for _, skill := range unit.Skills {
				skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
				require.NoError(t, err)

				for _, lesson := range skillDetail.Lessons {
					exercises, err := s.Courses().GetExercises(ctx, lesson.ID)
					require.NoError(t, err)

					// Each lesson should have the expected exercise count
					require.Equal(t, lesson.ExerciseCount, len(exercises),
						"Lesson %s level %d exercise count mismatch: expected %d, got %d",
						skill.Name, lesson.Level, lesson.ExerciseCount, len(exercises))
				}
			}
		}
	}
}

// Helper function to set up test store with seeded data
func setupTestStoreWithSeed(t *testing.T) store.Store {
	ctx := context.Background()

	s, err := sqlite.New(ctx, ":memory:")
	require.NoError(t, err)

	err = s.Ensure(ctx)
	require.NoError(t, err)

	err = s.SeedLanguages(ctx)
	require.NoError(t, err)

	err = s.SeedCourses(ctx)
	require.NoError(t, err)

	return s
}

// TestTotalDataCoverage provides a summary of total data coverage
func TestTotalDataCoverage(t *testing.T) {
	ctx := context.Background()
	s := setupTestStoreWithSeed(t)
	defer s.Close()

	var totalUnits, totalSkills, totalLessons, totalExercises int

	courses, err := s.Courses().ListCourses(ctx, "en")
	require.NoError(t, err)

	t.Logf("Total courses: %d", len(courses))

	for _, course := range courses {
		units, err := s.Courses().GetCoursePath(ctx, course.ID)
		require.NoError(t, err)

		totalUnits += len(units)

		for _, unit := range units {
			totalSkills += len(unit.Skills)

			for _, skill := range unit.Skills {
				skillDetail, err := s.Courses().GetSkill(ctx, skill.ID)
				require.NoError(t, err)

				totalLessons += len(skillDetail.Lessons)

				for _, lesson := range skillDetail.Lessons {
					exercises, err := s.Courses().GetExercises(ctx, lesson.ID)
					require.NoError(t, err)

					totalExercises += len(exercises)
				}
			}
		}
	}

	t.Logf("Total units: %d", totalUnits)
	t.Logf("Total skills: %d", totalSkills)
	t.Logf("Total lessons: %d", totalLessons)
	t.Logf("Total exercises: %d", totalExercises)

	// Verify minimum requirements
	require.GreaterOrEqual(t, len(courses), 4, "Should have at least 4 courses")
	require.GreaterOrEqual(t, totalUnits, 20, "Should have at least 20 units total")
	require.GreaterOrEqual(t, totalSkills, 40, "Should have at least 40 skills total")
	require.GreaterOrEqual(t, totalLessons, 200, "Should have at least 200 lessons total")
	require.GreaterOrEqual(t, totalExercises, 2000, "Should have at least 2000 exercises total")
}
