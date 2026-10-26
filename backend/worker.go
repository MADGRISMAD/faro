package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

type WorkItem struct {
	ID, Tenant, Site, Kind string
	Attempt                int
	Prompt                 Prompt
	Reserve                int64
}

func (a *App) claim(ctx context.Context) (*WorkItem, error) {
	tx, e := a.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var j WorkItem
	var keyword sql.NullString
	e = tx.QueryRowContext(ctx, `SELECT j.id,j.tenant_id,j.site_id,j.attempts,j.topic,j.business,j.audience,j.keyword,s.brand_description,s.tone,j.kind FROM jobs j JOIN sites s ON s.id=j.site_id WHERE j.status='queued' AND NOT EXISTS(SELECT 1 FROM jobs r WHERE r.tenant_id=j.tenant_id AND r.status='running') ORDER BY j.created_at FOR UPDATE OF j SKIP LOCKED LIMIT 1`).Scan(&j.ID, &j.Tenant, &j.Site, &j.Attempt, &j.Prompt.Topic, &j.Prompt.Business, &j.Prompt.Audience, &keyword, &j.Prompt.Brand, &j.Prompt.Tone, &j.Kind)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if keyword.Valid {
		j.Prompt.Keyword = keyword.String
	}
	j.Reserve = reserveAmount(a.Config)
	var result sql.Result
	result, e = tx.ExecContext(ctx, `UPDATE tenants SET reserved_microusd=reserved_microusd+$1 WHERE id=$2 AND budget_microusd-spent_microusd-reserved_microusd>=$1`, j.Reserve, j.Tenant)
	if e != nil {
		return nil, e
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		_, e = tx.ExecContext(ctx, `UPDATE jobs SET status='failed',error_message='Presupuesto insuficiente para reservar la generación',updated_at=now() WHERE id=$1`, j.ID)
		if e != nil {
			return nil, e
		}
		return nil, tx.Commit()
	}
	j.Attempt++
	_, e = tx.ExecContext(ctx, `UPDATE jobs SET status='running',attempts=$1,reserved_microusd=$2,updated_at=now() WHERE id=$3`, j.Attempt, j.Reserve, j.ID)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return &j, nil
}
func (a *App) recoverStale(ctx context.Context) error {
	tx, e := a.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, `SELECT id,tenant_id,reserved_microusd,attempts FROM jobs WHERE status='running' AND updated_at<now()-interval '2 minutes' FOR UPDATE SKIP LOCKED LIMIT 10`)
	if e != nil {
		return e
	}
	type stale struct {
		id, tenant string
		reserve    int64
		attempts   int
	}
	list := []stale{}
	for rows.Next() {
		var s stale
		if e = rows.Scan(&s.id, &s.tenant, &s.reserve, &s.attempts); e != nil {
			rows.Close()
			return e
		}
		list = append(list, s)
	}
	rows.Close()
	for _, s := range list {
		if _, e = tx.ExecContext(ctx, `UPDATE tenants SET reserved_microusd=reserved_microusd-$1,spent_microusd=spent_microusd+$2 WHERE id=$3`, s.reserve, s.reserve, s.tenant); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO usage_events(tenant_id,job_id,attempt,model,input_tokens,output_tokens,cost_microusd,status) VALUES($1,$2,$3,$4,0,0,$5,'unknown') ON CONFLICT(job_id,attempt) DO NOTHING`, s.tenant, s.id, s.attempts, a.Config.Model, s.reserve); e != nil {
			return e
		}
		status := "queued"
		if s.attempts >= 2 {
			status = "failed"
		}
		if _, e = tx.ExecContext(ctx, `UPDATE jobs SET status=$1,reserved_microusd=0,error_message='Worker interrumpido; se retuvo la reserva como costo incierto',updated_at=now() WHERE id=$2`, status, s.id); e != nil {
			return e
		}
	}
	return tx.Commit()
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(s)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "articulo"
	}
	if len(s) > 100 {
		s = s[:100]
		s = strings.TrimRight(s, "-")
	}
	return s
}
func (a *App) finish(ctx context.Context, j *WorkItem, p ProviderResult) error {
	tx, e := a.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var status string
	var attempt int
	e = tx.QueryRowContext(ctx, `SELECT status,attempts FROM jobs WHERE id=$1 FOR UPDATE`, j.ID).Scan(&status, &attempt)
	if e != nil {
		return e
	}
	if status != "running" || attempt != j.Attempt {
		return fmt.Errorf("job cambió de estado durante la llamada")
	}
	cost := actualCost(a.Config, p.InputTokens, p.OutputTokens)
	usageStatus := "measured"
	if p.Err != nil && p.InputTokens == 0 && p.OutputTokens == 0 && p.Unknown {
		cost = j.Reserve
		usageStatus = "unknown"
	}
	if cost > j.Reserve {
		return fmt.Errorf("uso mayor que reserva: requiere conciliación")
	}
	_, e = tx.ExecContext(ctx, `UPDATE tenants SET reserved_microusd=reserved_microusd-$1,spent_microusd=spent_microusd+$2 WHERE id=$3`, j.Reserve, cost, j.Tenant)
	if e != nil {
		return e
	}
	if cost > 0 {
		_, e = tx.ExecContext(ctx, `INSERT INTO usage_events(tenant_id,job_id,attempt,model,input_tokens,output_tokens,cost_microusd,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, j.Tenant, j.ID, j.Attempt, a.Config.Model, p.InputTokens, p.OutputTokens, cost, usageStatus)
		if e != nil {
			return e
		}
	}
	if p.Err != nil {
		jobStatus := "queued"
		if j.Attempt >= 2 {
			jobStatus = "failed"
		}
		_, e = tx.ExecContext(ctx, `UPDATE jobs SET status=$1,reserved_microusd=0,error_message=$2,updated_at=now() WHERE id=$3`, jobStatus, p.Err.Error(), j.ID)
		if e != nil {
			return e
		}
		return tx.Commit()
	}
	g := p.Article
	g.Content, e = sanitize(g.Content)
	if e != nil || !valid(g.Title, 180) || !valid(g.MetaDescription, 320) || !valid(g.Summary, 1000) {
		jobStatus := "queued"
		if j.Attempt >= 2 {
			jobStatus = "failed"
		}
		_, e = tx.ExecContext(ctx, `UPDATE jobs SET status=$1,reserved_microusd=0,error_message='Gemini devolvió contenido inválido',updated_at=now() WHERE id=$2`, jobStatus, j.ID)
		if e != nil {
			return e
		}
		return tx.Commit()
	}
	body, _ := json.Marshal(g.Content)
	articleSlug := slug(g.Title) + "-" + j.ID[:8]
	var article string
	e = tx.QueryRowContext(ctx, `INSERT INTO articles(tenant_id,site_id,job_id,title,slug,meta_description,summary,content,status,published_at) SELECT $1,$2,$3,$4,$5,$6,$7,$8,CASE WHEN s.autopilot THEN 'published' ELSE 'draft' END,CASE WHEN s.autopilot THEN now() END FROM sites s WHERE s.id=$2 RETURNING id`, j.Tenant, j.Site, j.ID, clean(g.Title, 180), articleSlug, clean(g.MetaDescription, 320), clean(g.Summary, 1000), body).Scan(&article)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `UPDATE jobs SET status='done',reserved_microusd=0,article_id=$1,error_message=NULL,updated_at=now() WHERE id=$2`, article, j.ID)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (a *App) Work(ctx context.Context) {
	lastRecovery, lastAutopilot := time.Time{}, time.Time{}
	for {
		if ctx.Err() != nil {
			return
		}
		if time.Since(lastRecovery) > time.Minute {
			if e := a.recoverStale(ctx); e != nil {
				slog.Error("Recuperación de jobs", "error", e)
			}
			lastRecovery = time.Now()
		}
		if time.Since(lastAutopilot) > 5*time.Minute {
			if e := a.autopilotTick(ctx); e != nil {
				slog.Error("Autopiloto", "error", e)
			}
			lastAutopilot = time.Now()
		}
		j, e := a.claim(ctx)
		if e != nil {
			slog.Error("Cola", "error", e)
			time.Sleep(2 * time.Second)
			continue
		}
		if j == nil {
			time.Sleep(700 * time.Millisecond)
			continue
		}
		if j.Kind == "research" {
			a.runResearch(ctx, j)
			continue
		}
		callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		p := a.Provider.Create(callCtx, j.Prompt, a.Config)
		cancel()
		if e = a.finish(ctx, j, p); e != nil {
			slog.Error("No se pudo completar el job", "error", e)
		}
	}
}
