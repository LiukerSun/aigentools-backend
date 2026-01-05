package services

import (
	"aigentools-backend/internal/database"
	"aigentools-backend/internal/models"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.Migrator().DropTable(&models.AIModel{})
	err = db.AutoMigrate(&models.AIModel{})
	if err != nil {
		panic("failed to migrate database")
	}

	database.DB = db
}

func setupTestRedis() *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	database.RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr
}

func TestGetAllModelsSimple(t *testing.T) {
	setupTestDB()

	// Seed data
	modelsList := []models.AIModel{
		{Name: "GPT-4", Status: models.AIModelStatusOpen, Description: "Desc 1"},
		{Name: "Claude-3", Status: models.AIModelStatusOpen, Description: "Desc 2"},
		{Name: "Llama-2", Status: models.AIModelStatusDraft, Description: "Desc 3"},
	}
	database.DB.Create(&modelsList)

	simpleModels, err := GetAllModelsSimple()
	assert.NoError(t, err)
	assert.Len(t, simpleModels, 3)

	for _, m := range simpleModels {
		assert.NotEmpty(t, m.Name)
		assert.NotEmpty(t, m.Status)
		assert.NotEmpty(t, m.Description)
		assert.Empty(t, m.Parameters) // Parameters should be empty/nil
	}
}

func TestGetModelParametersByID(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()

	params := models.JSON{
		"request_header":      []interface{}{},
		"request_body":        []interface{}{},
		"response_parameters": []interface{}{},
	}

	model := models.AIModel{
		Name:       "Test Model",
		Parameters: params,
	}
	database.DB.Create(&model)

	// Test Cache Miss (fetch from DB)
	fetchedParams, err := GetModelParametersByID(model.ID)
	assert.NoError(t, err)

	// Convert both to JSON string for comparison to avoid map type issues (e.g. []interface{} vs []string)
	// or use assert.EqualValues
	assert.Equal(t, params, fetchedParams)

	// Verify it's in Redis now
	// We can try to modify the DB and fetch again. If it returns the old value, it's from cache.

	// Modify DB
	newParams := models.JSON{
		"request_header": []interface{}{"modified"},
	}
	// Bypass GORM hooks if any, but direct update should be fine
	database.DB.Model(&model).Update("Parameters", newParams)

	// Fetch again - should get OLD params from cache
	cachedParams, err := GetModelParametersByID(model.ID)
	assert.NoError(t, err)
	assert.Equal(t, params, cachedParams)
	assert.NotEqual(t, newParams, cachedParams)

	// Fast forward time to expire cache
	mr.FastForward(time.Hour + time.Minute)

	// Fetch again - should get NEW params from DB
	freshParams, err := GetModelParametersByID(model.ID)
	assert.NoError(t, err)

	// Note: unmarshaling from DB might have slight type diffs (e.g. "modified" string in interface{})
	// We expect "freshParams" to match "newParams" content-wise
	// However, newParams was created as map[string]interface{}.
	// When stored in DB and retrieved, it should be similar.
	// But since newParams structure is loose here (missing other fields),
	// and models.JSON is just a map, it should work.

	// Wait, models.ValidateModelParameters enforces structure?
	// The service function GetModelParametersByID does NOT call ValidateModelParameters.
	// So we can save whatever we want in the test.

	// Verify content of freshParams
	header, ok := freshParams["request_header"].([]interface{})
	if assert.True(t, ok) {
		assert.Equal(t, "modified", header[0])
	}
}

func TestUpdateAIModelClearsCache(t *testing.T) {
	setupTestDB()
	mr := setupTestRedis()
	defer mr.Close()

	params := models.JSON{
		"request_header":      []interface{}{},
		"request_body":        []interface{}{},
		"response_parameters": []interface{}{},
	}

	model := models.AIModel{
		Name:       "Cache Test Model",
		Parameters: params,
	}
	database.DB.Create(&model)

	// 1. Populate cache
	_, err := GetModelParametersByID(model.ID)
	assert.NoError(t, err)

	// 2. Update model using service
	newParams := models.JSON{
		"request_header": []interface{}{
			map[string]interface{}{"name": "Authorization", "type": "string", "required": true, "description": "Token", "example": "Bearer"},
		},
		"request_body":        []interface{}{},
		"response_parameters": []interface{}{},
	}
	model.Parameters = newParams

	err = UpdateAIModel(&model)
	assert.NoError(t, err)

	// 3. Fetch again - should get NEW params
	fetchedParams, err := GetModelParametersByID(model.ID)
	assert.NoError(t, err)

	header, ok := fetchedParams["request_header"].([]interface{})
	if assert.True(t, ok) {
		// If cache is NOT cleared, we get empty list from old params
		assert.NotEmpty(t, header, "Expected updated parameters, but got empty header (likely from stale cache)")
	}
}
