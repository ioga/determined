//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestAddJob(t *testing.T) {
	closeDB := setupDBForTest(t)
	defer closeDB()

	t.Run("add job", func(t *testing.T) {
		_, err := createAndAddJob(0)
		require.NoError(t, err)
	})

	t.Run("add job with duplicate id", func(t *testing.T) {
		j, err := createAndAddJob(0)
		require.NoError(t, err)

		// change job type and re-add job
		j.JobType = model.JobTypeExperiment
		err = AddJob(&j)
		require.Error(t, err)
	})

	t.Run("add job with no job type", func(t *testing.T) {
		err := AddJob(&model.Job{JobID: model.NewJobID()})
		require.Error(t, err)
	})
}

func TestJobByID(t *testing.T) {
	closeDB := setupDBForTest(t)
	defer closeDB()

	t.Run("add and retrieve job", func(t *testing.T) {
		// create and send job
		sendJob, err := createAndAddJob(10)
		require.NoError(t, err)

		// retrieve job and test for equality
		recvJob, err := JobByID(context.Background(), sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, sendJob.JobID, recvJob.JobID)
		assert.Equal(t, sendJob.JobType, recvJob.JobType)
		assert.Equal(t, sendJob.OwnerID, recvJob.OwnerID)
		assert.Equal(t, sendJob.QPos.Equal(recvJob.QPos), true)
	})

	t.Run("retrieve non-existent job", func(t *testing.T) {
		// attempt to retrieve job that does not exist
		recvJob, err := JobByID(context.Background(), model.NewJobID())
		require.Error(t, err)
		require.Nil(t, recvJob)
	})
}

func TestUpdateJobPosition(t *testing.T) {
	closeDB := setupDBForTest(t)
	defer closeDB()

	t.Run("update position", func(t *testing.T) {
		// create and send job
		sendJob, err := createAndAddJob(10)
		require.NoError(t, err)

		// update job position
		newPos := decimal.NewFromInt(5)
		err = UpdateJobPosition(context.Background(), sendJob.JobID, newPos)
		require.NoError(t, err)

		// retrieve job and confirm pos update
		recvJob, err := JobByID(context.Background(), sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, newPos.Equal(recvJob.QPos), true)
		// expect other fields to remain the same
		assert.Equal(t, sendJob.JobID, recvJob.JobID)
		assert.Equal(t, sendJob.JobType, recvJob.JobType)
		assert.Equal(t, sendJob.OwnerID, recvJob.OwnerID)
	})

	t.Run("update position - negative value", func(t *testing.T) {
		// create and send job
		sendJob, err := createAndAddJob(10)
		require.NoError(t, err)

		// update job position
		newPos := decimal.NewFromInt(-5)
		err = UpdateJobPosition(context.Background(), sendJob.JobID, newPos)
		require.NoError(t, err)

		// retrieve job and confirm pos update
		recvJob, err := JobByID(context.Background(), sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, newPos.Equal(recvJob.QPos), true)
		// expect other fields to remain the same
		assert.Equal(t, sendJob.JobID, recvJob.JobID)
		assert.Equal(t, sendJob.JobType, recvJob.JobType)
		assert.Equal(t, sendJob.OwnerID, recvJob.OwnerID)
	})

	t.Run("update position - empty ID", func(t *testing.T) {
		sendJob, err := createAndAddJob(10)
		require.NoError(t, err)

		// update job position
		newPos := decimal.NewFromInt(5)
		err = UpdateJobPosition(context.Background(), model.JobID(""), newPos)
		require.Error(t, err)

		// retrieve job and ensure queue pos not updated
		recvJob, err := JobByID(context.Background(), sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, sendJob.QPos.Equal(recvJob.QPos), true)
	})

	t.Run("update position - ID does not exist", func(t *testing.T) {
		// create and send job
		_, err := createAndAddJob(10)
		require.NoError(t, err)

		// update job position for a job that doesn't exist
		newPos := decimal.NewFromInt(5)
		err = UpdateJobPosition(context.Background(), model.NewJobID(), newPos)
		require.NoError(t, err)
	})
}

// TODO [RM-27] initialize db in a TestMain(...) when there's enough package isolation.
func setupDBForTest(t *testing.T) func() {
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closeDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	return closeDB
}

func createAndAddJob(pos int64) (model.Job, error) {
	sendJob := model.Job{
		JobID:   model.NewJobID(),
		JobType: model.JobTypeExperiment,
		QPos:    decimal.NewFromInt(pos),
	}
	err := AddJob(&sendJob)
	return sendJob, err
}
