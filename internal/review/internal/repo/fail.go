package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/KNICEX/InkFlow/internal/review/internal/domain"
	"github.com/KNICEX/InkFlow/internal/review/internal/event"
	"github.com/KNICEX/InkFlow/internal/review/internal/repo/dao"
)

var (
	ErrUnknownReviewType = errors.New("unknown review type")
)

type ReviewFailRepo interface {
	Create(ctx context.Context, evt domain.FailReview, er error) error
	Find(ctx context.Context, offset, limit int) ([]domain.FailReview, error)
	Delete(ctx context.Context, ids []int64) error
}

type reviewFailRepo struct {
	dao dao.ReviewFailDAO
}

func NewReviewFailRepo(dao dao.ReviewFailDAO) ReviewFailRepo {
	return &reviewFailRepo{
		dao: dao,
	}
}

func (r *reviewFailRepo) Create(ctx context.Context, evt domain.FailReview, er error) error {
	eventJson, err := json.Marshal(evt.Event)
	if err != nil {
		return fmt.Errorf("marshal review event failed: %w", err)
	}

	record := dao.ReviewFail{
		Type:  string(evt.Type),
		Event: string(eventJson),
		Error: er.Error(),
	}

	return r.dao.Insert(ctx, record)
}

func (r *reviewFailRepo) Find(ctx context.Context, offset, limit int) ([]domain.FailReview, error) {
	records, err := r.dao.Find(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	res := make([]domain.FailReview, 0, len(records))
	for _, rec := range records {
		domainRec, err := r.toDomain(&rec)
		if err != nil {
			return nil, err
		}
		res = append(res, domainRec)
	}

	return res, nil
}

func (r *reviewFailRepo) Delete(ctx context.Context, ids []int64) error {
	return r.dao.Delete(ctx, ids)
}

func (r *reviewFailRepo) toDomain(entity *dao.ReviewFail) (domain.FailReview, error) {
	reviewType := domain.ReviewType(entity.Type)
	payload := []byte(entity.Event)

	// Backward compatibility: old records stored {Type, Event, ...} into Event column and left Type empty.
	var legacy struct {
		Type  domain.ReviewType `json:"Type"`
		Event json.RawMessage   `json:"Event"`
	}
	if err := json.Unmarshal(payload, &legacy); err == nil && len(legacy.Event) > 0 {
		if reviewType == "" {
			reviewType = legacy.Type
		}
		if strings.TrimSpace(string(legacy.Event)) != "null" {
			payload = legacy.Event
		}
	}

	switch reviewType {
	case domain.ReviewTypeInk:
		var evt event.ReviewInkEvent
		if err := json.Unmarshal(payload, &evt); err != nil {
			return domain.FailReview{}, err
		}
		if strings.TrimSpace(evt.WorkflowId) == "" {
			return domain.FailReview{}, errors.New("invalid fail review event: empty workflowId")
		}
		return domain.FailReview{
			Id:        entity.Id,
			Type:      reviewType,
			Event:     evt,
			Error:     errors.New(entity.Error),
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
		}, nil
	default:
		return domain.FailReview{}, fmt.Errorf("%w : %s", ErrUnknownReviewType, reviewType)
	}
}
