package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/fatalistix/crack-hash-worker/internal/config"
	"github.com/fatalistix/crack-hash-worker/internal/domain/model"
	"github.com/fatalistix/crack-hash-worker/internal/http/client"
	"github.com/fatalistix/slogattr"
	"log/slog"
	"sync"
	"time"
)

type CrackService struct {
	wg           *sync.WaitGroup
	parts        chan<- model.Part
	results      chan<- model.CompletedPart
	workersCount uint64
	log          *slog.Logger
}

func NewCrackService(
	log *slog.Logger,
	managerConfig config.ManagerConfig,
	workerConfig config.WorkerConfig,
	workerId string,
) *CrackService {
	wg := new(sync.WaitGroup)
	parts := make(chan model.Part)
	results := make(chan model.CompletedPart)

	for i := uint64(0); i < workerConfig.GoroutineCount; i++ {
		wg.Add(1)
		logWithGoroutineId := log.With(slog.Uint64("goroutine worker id", i))
		go worker(logWithGoroutineId, parts, results, workerConfig.SubTaskTimeout, wg)
	}

	log.Info("worker pool created", slog.Uint64("workers count", workerConfig.GoroutineCount))

	go resultHandler(log, managerConfig.Address, workerId, results, workerConfig.GoroutineCount)

	log.Info("result handler started")

	return &CrackService{
		wg:           wg,
		parts:        parts,
		results:      results,
		workersCount: workerConfig.GoroutineCount,
		log:          log,
	}
}

func worker(log *slog.Logger, parts <-chan model.Part, results chan<- model.CompletedPart, subTaskTimeout time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()
	for part := range parts {
		log.Info("worker is processing part", slog.Any("part", part))
		handlePart(log, part, results, subTaskTimeout)
	}
}

func resultHandler(log *slog.Logger, managerAddress, workerId string, results <-chan model.CompletedPart, workersCount uint64) {
	const op = "service.resultHandler"

	type partWithCount struct {
		Part   model.CompletedPart
		Count  uint64
		Errors []error
	}

	log = log.With(
		slog.String("op", op),
	)

	idToResults := make(map[string]partWithCount)

	completer := client.NewCompleter(log)
	for result := range results {

		log.Info("serving partial result", slog.Any("part", result))

		value, ok := idToResults[result.TaskId]
		if !ok {
			errors := make([]error, 0)
			if result.Error != nil {
				log.Error("error during computation", slogattr.Err(result.Error))
				errors = append(errors, result.Error)
			}
			value = partWithCount{Part: result, Count: 1, Errors: errors}
			log.Info("first part of result", slog.String("task_id", result.TaskId))
		} else {
			value.Part.Start = min(result.Start, value.Part.Start)
			value.Part.End = max(result.End, value.Part.End)
			value.Part.Data = append(value.Part.Data, result.Data...)
			if result.Error != nil {
				log.Error("error during computation", slogattr.Err(result.Error))
				value.Errors = append(value.Errors, result.Error)
			}
			value.Count++
			log.Info("part of result", slog.String("task_id", result.TaskId))
		}

		if value.Count < workersCount {
			idToResults[result.TaskId] = value
			log.Info("current partial result", slog.Any("partial result", value.Part))
			continue
		}

		log.Info("full result", slog.String("task_id", result.TaskId))

		delete(idToResults, result.TaskId)

		if len(value.Errors) > 0 {
			log.Error("errors during computation, result won't be sent", slog.Any("errors", value.Errors))
			continue
		}

		request := client.CompleteRequest{
			RequestId: result.RequestId,
			TaskId:    result.TaskId,
			WorkerId:  workerId,
			Start:     value.Part.Start,
			End:       value.Part.End,
			Data:      value.Part.Data,
		}
		if err := completer.Complete(managerAddress, request); err != nil {
			log.Error("failed to complete", slogattr.Err(err))
		}
	}
}

func handlePart(log *slog.Logger, part model.Part, results chan<- model.CompletedPart, subTaskTimeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), subTaskTimeout)
	defer cancel()

	generator := NewPermutationGenerator(part.Alphabet, part.Start, part.End-part.Start)
	result := make([]string, 0)
	var err error = nil

outer:
	for generator.HasNext() {
		select {
		case <-ctx.Done():
			{
				err = ctx.Err()
				break outer
			}
		default:
			{
				value := generator.Next()
				log.Debug("generated value", slog.Any("value", value))
				hashBytes := md5.Sum([]byte(value))
				hash := hex.EncodeToString(hashBytes[:])
				if hash == part.Hash {
					result = append(result, value)
				}
			}
		}
	}

	completedPart := model.CompletedPart{
		RequestId: part.RequestId,
		TaskId:    part.TaskId,
		Data:      result,
		Start:     part.Start,
		End:       part.End,
		Error:     err,
	}

	log.Info("completed part", slog.Any("completed part", completedPart))
	results <- completedPart
}

func (s *CrackService) StartTask(task model.Task) {
	s.log.Info("starting task", slog.Any("task", task))

	separated := s.separate(task)

	s.log.Info("separated task", slog.Any("parts", separated))

	for _, part := range separated {
		s.parts <- part
	}
}

func (s *CrackService) separate(task model.Task) []model.Part {
	separated := make([]model.Part, s.workersCount)
	totalSize := task.End - task.Start
	partSize := totalSize / s.workersCount

	start := task.Start

	for i := uint64(0); i < s.workersCount; i++ {
		part := model.Part{
			RequestId: task.RequestId,
			TaskId:    task.TaskId,
			Alphabet:  task.Alphabet,
			Hash:      task.Hash,
			MaxLength: task.MaxLength,
			Start:     start,
			End:       start + partSize,
		}

		separated[i] = part

		start += partSize
	}

	separated[s.workersCount-1].End = task.End

	return separated
}

func (s *CrackService) Close() error {
	const op = "service.CrackService.Close"

	log := s.log.With(
		slog.String("operation", op),
	)

	log.Info("stopping...")

	close(s.parts)
	s.wg.Wait()
	close(s.results)

	log.Info("stopped")

	return nil
}
