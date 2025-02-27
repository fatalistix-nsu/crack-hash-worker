package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatalistix/slogattr"
	"log/slog"
	"net/http"
)

const completePath = basePath + "/request"

type CompleteRequest struct {
	RequestId string   `json:"request_id"`
	TaskId    string   `json:"task_id"`
	WorkerId  string   `json:"worker_id"`
	Start     uint64   `json:"start"`
	End       uint64   `json:"end"`
	Data      []string `json:"data"`
}

type Completer struct {
	log *slog.Logger
}

func NewCompleter(log *slog.Logger) *Completer {
	return &Completer{
		log: log,
	}
}

func (c *Completer) Complete(managerAddress string, request CompleteRequest) error {
	const op = "http.client.Completer.Complete"

	log := c.log.With(
		slog.String("op", op),
	)

	log.Debug("sending complete request", slog.Any("request", request))

	requestBytes, err := json.Marshal(request)
	if err != nil {
		log.Error("error marshaling bytes", slogattr.Err(err))
		return fmt.Errorf("%s: error marshaling request %w", op, err)
	}

	httpRequest, err := http.NewRequest(http.MethodPatch, makeUrl(managerAddress, completePath), bytes.NewBuffer(requestBytes))
	if err != nil {
		log.Error("error creating http request", slogattr.Err(err))
		return fmt.Errorf("%s: error creating http request %w", op, err)
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		log.Error("error executing http request", slogattr.Err(err))
		return fmt.Errorf("%s: error executing http request %w", op, err)
	}

	if httpResponse.StatusCode != http.StatusAccepted {
		log.Error("error completing task: unexpected http status code", slog.Int("status code", httpResponse.StatusCode))
		return fmt.Errorf("%s: error completing task: unexpected http status code: status %d", op, httpResponse.StatusCode)
	}

	log.Info("task completed successfully", slog.Any("request", request))

	return nil
}
