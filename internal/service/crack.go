package service

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/fatalistix/crack-hash-worker/internal/domain/model"
	"github.com/fatalistix/crack-hash-worker/internal/http/client"
	"github.com/fatalistix/slogattr"
	"log/slog"
	"runtime"
	"sync"
)

type CrackService struct {
	wg           *sync.WaitGroup
	parts        chan<- model.Part
	workersCount uint64
	log          *slog.Logger
}

func NewCrackService(log *slog.Logger, managerAddress, workerId string) *CrackService {
	wg := new(sync.WaitGroup)
	parts := make(chan model.Part)
	results := make(chan model.CompletedPart)
	numCpu := runtime.NumCPU()
	//numCpu := 1

	for i := 0; i < numCpu; i++ {
		wg.Add(1)
		logWithGoroutineId := log.With(slog.Int("goroutine worker id", i))
		go worker(logWithGoroutineId, parts, results, wg)
	}

	log.Info("worker pool created", slog.Int("workers count", numCpu))

	go resultHandler(log, managerAddress, workerId, results, uint64(numCpu))

	log.Info("result handler started")

	return &CrackService{
		wg:           wg,
		parts:        parts,
		workersCount: uint64(numCpu),
		log:          log,
	}
}

func worker(log *slog.Logger, parts <-chan model.Part, results chan<- model.CompletedPart, wg *sync.WaitGroup) {
	defer wg.Done()
	for part := range parts {
		log.Info("worker is processing part", slog.Any("part", part))
		handlePart(log, part, results)
	}
}

func resultHandler(log *slog.Logger, managerAddress, workerId string, results <-chan model.CompletedPart, workersCount uint64) {
	type partWithCount struct {
		Part  model.CompletedPart
		Count uint64
	}

	idToResults := make(map[string]partWithCount)

	completer := client.NewCompleter(log)
	for result := range results {

		log.Info("serving partial result", slog.Any("part", result))

		value, ok := idToResults[result.TaskId]
		if !ok {
			value = partWithCount{Part: result, Count: 1}
			log.Info("first part of result", slog.String("task_id", result.TaskId))
		} else {
			value.Part.Start = min(result.Start, value.Part.Start)
			value.Part.End = max(result.End, value.Part.End)
			value.Part.Data = append(value.Part.Data, result.Data...)
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

func handlePart(log *slog.Logger, part model.Part, results chan<- model.CompletedPart) {
	generator := NewPermutationGenerator(part.Alphabet, part.Start, part.End-part.Start)
	result := make([]string, 0)
	for generator.HasNext() {
		value := generator.Next()
		log.Debug("generated value", slog.Any("value", value))
		hashBytes := md5.Sum([]byte(value))
		hash := hex.EncodeToString(hashBytes[:])
		if hash == part.Hash {
			result = append(result, value)
		}
	}

	completedPart := model.CompletedPart{
		RequestId: part.RequestId,
		TaskId:    part.TaskId,
		Data:      result,
		Start:     part.Start,
		End:       part.End,
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
