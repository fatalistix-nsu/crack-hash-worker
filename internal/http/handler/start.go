package handler

import (
	"fmt"
	"github.com/fatalistix/crack-hash-worker/internal/domain/model"
	"github.com/labstack/echo/v4"
	"net/http"
)

type TaskStarter interface {
	StartTask(model.Task)
}

type TaskRequest struct {
	RequestId string `json:"request_id" validate:"required"`
	TaskId    string `json:"task_id" validate:"required"`
	Alphabet  string `json:"alphabet" validate:"required,uniquechars"`
	Hash      string `json:"hash" validate:"required,md5hash"`
	MaxLength uint64 `json:"max_length" validate:"required,min=1"`
	Start     uint64 `json:"start" validate:"min=0"`
	End       uint64 `json:"end" validate:"required,gtfield=Start"`
}

func MakeStartTaskHandlerFunc(taskStarter TaskStarter) echo.HandlerFunc {
	return func(c echo.Context) error {
		var request TaskRequest

		if err := c.Bind(&request); err != nil {
			return echo.NewHTTPError(http.StatusUnsupportedMediaType, "invalid request body").SetInternal(err)
		}

		if err := c.Validate(request); err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %s", err.Error())).SetInternal(err)
		}

		task := MapRequestToModel(request)
		taskStarter.StartTask(task)

		return c.JSON(http.StatusOK, nil)
	}
}

func MapRequestToModel(request TaskRequest) model.Task {
	return model.Task{
		RequestId: request.RequestId,
		TaskId:    request.TaskId,
		Alphabet:  request.Alphabet,
		Hash:      request.Hash,
		MaxLength: request.MaxLength,
		Start:     request.Start,
		End:       request.End,
	}
}
