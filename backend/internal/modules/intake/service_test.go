package intake

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectService_Create(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	proj, err := svc.Create("user-1", "前端工程师简历")
	assert.NoError(t, err)
	assert.Equal(t, "前端工程师简历", proj.Title)
	assert.Equal(t, "user-1", proj.UserID)
	assert.Equal(t, "active", proj.Status)
	assert.Greater(t, proj.ID, uint(0))
}

func TestProjectService_List_FiltersByUserID(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	svc.Create("user-1", "项目A")
	svc.Create("user-2", "项目B")
	svc.Create("user-1", "项目C")

	projects, err := svc.List("user-1")
	assert.NoError(t, err)
	assert.Len(t, projects, 2)
}

func TestProjectService_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "测试项目")

	proj, err := svc.GetByID("user-1", created.ID)
	assert.NoError(t, err)
	assert.Equal(t, "测试项目", proj.Title)
}

func TestProjectService_GetByID_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	_, err := svc.GetByID("user-1", 9999)
	assert.Error(t, err)
}

func TestProjectService_GetByID_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "我的项目")

	_, err := svc.GetByID("user-2", created.ID)
	assert.Error(t, err)
}

func TestProjectService_Delete(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "待删除")

	err := svc.Delete("user-1", created.ID)
	assert.NoError(t, err)

	_, err = svc.GetByID("user-1", created.ID)
	assert.Error(t, err)
}

func TestProjectService_Delete_WrongUser(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewProjectService(db)

	created, _ := svc.Create("user-1", "我的项目")

	err := svc.Delete("user-2", created.ID)
	assert.Error(t, err)
}
