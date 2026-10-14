package scheduler

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/fx"

	"sslplatform/internal/common/apperror"
	"sslplatform/internal/etl"
	"sslplatform/internal/paging"
	"sslplatform/internal/task"
)

type Service struct {
	repo    *Repository
	etl     *etl.ETLService
	cron    *cron.Cron
	mu      sync.Mutex
	entries map[uint]cron.EntryID
	bound   map[uint]string
	stop    chan struct{}
}

func NewService(lc fx.Lifecycle, repo *Repository, pipelines *etl.ETLService) *Service {
	svc := &Service{
		repo:    repo,
		etl:     pipelines,
		cron:    cron.New(),
		entries: map[uint]cron.EntryID{},
		bound:   map[uint]string{},
		stop:    make(chan struct{}),
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := svc.Reload(ctx); err != nil {
				return err
			}
			svc.cron.Start()
			go svc.loop()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			close(svc.stop)
			done := svc.cron.Stop()
			select {
			case <-done.Done():
			case <-ctx.Done():
			}
			return nil
		},
	})
	return svc
}

func (s *Service) Create(ctx context.Context, tenantID uint, req CreateRequest) (*View, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperror.New(400, "请填写调度名称")
	}
	if _, err := parseCron(req.Cron); err != nil {
		return nil, err
	}
	if _, err := s.etl.DetailJob(ctx, tenantID, req.ETLJobID); err != nil {
		return nil, err
	}
	next := nextRun(req.Cron)
	item := &Schedule{
		TenantID: tenantID, Name: strings.TrimSpace(req.Name), ETLJobID: req.ETLJobID,
		Cron: strings.TrimSpace(req.Cron), Enabled: req.Enabled, NextRunAt: next,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	_ = s.Reload(ctx)
	return toView(item), nil
}

func (s *Service) Update(ctx context.Context, tenantID uint, req UpdateRequest) (*View, error) {
	item, err := s.repo.Get(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		item.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Cron) != "" {
		if _, err := parseCron(req.Cron); err != nil {
			return nil, err
		}
		item.Cron = strings.TrimSpace(req.Cron)
		item.NextRunAt = nextRun(item.Cron)
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	_ = s.Reload(ctx)
	return toView(item), nil
}

func (s *Service) Delete(ctx context.Context, tenantID, id uint) error {
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	_ = s.Reload(ctx)
	return nil
}

func (s *Service) Enable(ctx context.Context, tenantID, id uint, enabled bool) (*View, error) {
	return s.Update(ctx, tenantID, UpdateRequest{ID: id, Enabled: &enabled})
}

func (s *Service) List(ctx context.Context, tenantID uint, req ListRequest) ([]View, int, int, int64, error) {
	page, size := paging.Norm(req.Page, req.Size)
	rows, total, err := s.repo.List(ctx, tenantID, page, size, req.Filters)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	views := make([]View, 0, len(rows))
	for i := range rows {
		views = append(views, *toView(&rows[i]))
	}
	return views, page, size, total, nil
}

func (s *Service) Reload(ctx context.Context) error {
	rows, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return err
	}
	want := map[uint]Schedule{}
	for _, row := range rows {
		want[row.ID] = row
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, entry := range s.entries {
		row, ok := want[id]
		if !ok || s.bound[id] != row.Cron {
			s.cron.Remove(entry)
			delete(s.entries, id)
			delete(s.bound, id)
		}
	}
	for id, row := range want {
		if _, ok := s.entries[id]; ok {
			continue
		}
		scheduleID := row.ID
		entryID, err := s.cron.AddFunc(row.Cron, func() { s.fire(scheduleID) })
		if err != nil {
			slog.Error("添加调度失败", "id", row.ID, "err", err)
			continue
		}
		s.entries[id] = entryID
		s.bound[id] = row.Cron
	}
	return nil
}

func (s *Service) fire(id uint) {
	ctx := context.Background()
	item, err := s.repo.GetByID(ctx, id)
	if err != nil || !item.Enabled {
		return
	}
	_, err = s.etl.Execute(ctx, item.TenantID, 0, etl.JobExecuteRequest{ID: item.ETLJobID, Force: false}, task.TriggerSchedule, false)
	status := "triggered"
	if err != nil {
		status = "failed"
		slog.Error("调度执行失败", "id", id, "err", err)
	}
	now := time.Now()
	_ = s.repo.MarkRun(ctx, id, now, nextRun(item.Cron), status)
}

func (s *Service) loop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			if err := s.Reload(context.Background()); err != nil {
				slog.Error("刷新调度失败", "err", err)
			}
		}
	}
}

func parseCron(expr string) (cron.Schedule, error) {
	parsed, err := cron.ParseStandard(strings.TrimSpace(expr))
	if err != nil {
		return nil, apperror.New(400, "Cron 表达式无效，请使用 5 段格式，例如 15 3 * * *")
	}
	return parsed, nil
}

func nextRun(expr string) *time.Time {
	parsed, err := parseCron(expr)
	if err != nil {
		return nil
	}
	next := parsed.Next(time.Now())
	return &next
}

func toView(item *Schedule) *View {
	return &View{
		ID: item.ID, Name: item.Name, ETLJobID: item.ETLJobID, Cron: item.Cron, Enabled: item.Enabled,
		LastRunAt: item.LastRunAt, NextRunAt: item.NextRunAt, LastStatus: item.LastStatus, CreatedAt: item.CreatedAt,
	}
}
