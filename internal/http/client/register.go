package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatalistix/slogattr"
	"io"
	"log/slog"
	"net/http"
)

const registerPath = basePath + "/register"

type RegisterRequest struct {
	WorkerPort int `json:"worker_port"`
}

type RegisterResponse struct {
	WorkerId string `json:"worker_id"`
}

type Registerer struct {
	log *slog.Logger
}

func NewRegisterer(log *slog.Logger) *Registerer {
	return &Registerer{
		log: log,
	}
}

func (r *Registerer) Register(managerAddress string, workerPort int) (string, error) {
	const op = "http.client.Registerer.Register"

	log := r.log.With(
		slog.String("op", op),
	)

	request := RegisterRequest{
		WorkerPort: workerPort,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		log.Error("error marshaling register request", slog.Any("request", request), slogattr.Err(err))
		return "", fmt.Errorf("%s: error marshaling request: %w", op, err)
	}

	httpResponse, err := http.Post(makeUrl(managerAddress, registerPath), "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		log.Error("error sending request", slog.Any("request", request), slogattr.Err(err))
		return "", fmt.Errorf("%s: error sending request: %w", op, err)
	}

	defer r.closeOrLog(httpResponse.Body)

	if httpResponse.StatusCode != http.StatusAccepted {
		log.Error("error registering worker: unexpected status code", slog.Any("request", request), slog.Int("status code", httpResponse.StatusCode))
		return "", fmt.Errorf("%s: error registering worker: unexpected status code %s", op, httpResponse.Status)
	}

	var response RegisterResponse

	decoder := json.NewDecoder(httpResponse.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&response); err != nil {
		log.Error("error decoding response body", slogattr.Err(err))
		return "", fmt.Errorf("%s: error decoding response body: %w", op, err)
	}

	return response.WorkerId, nil
}

func (r *Registerer) closeOrLog(closer io.Closer) {
	const op = "http.client.close"

	if err := closer.Close(); err != nil {
		r.log.Error(
			"unable to close",
			slog.String("op", op),
			slog.Any("closer", closer),
			slogattr.Err(err),
		)
	}
}
