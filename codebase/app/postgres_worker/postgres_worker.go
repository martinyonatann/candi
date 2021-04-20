package postgresworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lib/pq"
	"pkg.agungdp.dev/candi/candihelper"
	"pkg.agungdp.dev/candi/candiutils"
	"pkg.agungdp.dev/candi/codebase/factory"
	"pkg.agungdp.dev/candi/codebase/factory/types"
	"pkg.agungdp.dev/candi/config/env"
	"pkg.agungdp.dev/candi/logger"
	"pkg.agungdp.dev/candi/tracer"
)

/*
Postgres Worker
Listen event from data change from selected table in postgres
*/

var (
	shutdown, semaphore, startWorkerCh, releaseWorkerCh chan struct{}
)

type (
	postgresWorker struct {
		consul    *candiutils.Consul
		isHaveJob bool
		listener  *pq.Listener
		handlers  map[string]types.WorkerHandlerFunc
		wg        sync.WaitGroup
	}
)

// NewWorker create new redis subscriber
func NewWorker(service factory.ServiceFactory) factory.AppServerFactory {
	worker := new(postgresWorker)
	shutdown, semaphore = make(chan struct{}, 1), make(chan struct{}, env.BaseEnv().MaxGoroutines)

	worker.handlers = make(map[string]types.WorkerHandlerFunc)
	db, listener := getListener()
	execCreateFunctionEventQuery(db)

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(types.PostgresListener); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				logger.LogYellow(fmt.Sprintf(`[POSTGRES-LISTENER] (table): %-15s  --> (module): "%s"`, `"`+handler.Pattern+`"`, m.Name()))
				worker.handlers[handler.Pattern] = handler.HandlerFunc
				execTriggerQuery(db, handler.Pattern)
			}
		}
	}

	if len(worker.handlers) == 0 {
		log.Println("postgres listener: no table event provided")
	} else {
		fmt.Printf("\x1b[34;1m⇨ Postgres Event Listener running with %d table\x1b[0m\n\n", len(worker.handlers))
	}

	worker.listener = listener
	return worker
}

func (p *postgresWorker) Serve() {
	for {
		select {
		case e := <-p.listener.Notify:

			semaphore <- struct{}{}
			p.wg.Add(1)
			go func(event *pq.Notification) {
				defer func() {
					p.wg.Done()
					<-semaphore
				}()

				trace := tracer.StartTrace(context.Background(), "PostgresEventListener")
				ctx := trace.Context()
				defer func() {
					if r := recover(); r != nil {
						tracer.SetError(ctx, fmt.Errorf("panic: %v", r))
					}
					logger.LogGreen("postgres_listener > trace_url: " + tracer.GetTraceURL(ctx))
					trace.Finish()
				}()

				var eventPayload EventPayload
				json.Unmarshal([]byte(event.Extra), &eventPayload)

				trace.SetTag("database", candihelper.MaskingPasswordURL(env.BaseEnv().DbSQLWriteDSN))
				trace.SetTag("table_name", eventPayload.Table)
				trace.SetTag("action", eventPayload.Action)
				trace.Log("payload", event.Extra)
				if err := p.handlers[eventPayload.Table](trace.Context(), []byte(event.Extra)); err != nil {
					trace.SetError(err)
				}
			}(e)

		case <-time.After(2 * time.Minute):
			p.listener.Ping()
		}
	}
}

func (p *postgresWorker) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping Postgres Event Listener worker...\x1b[0m")
	defer func() {
		if p.consul != nil {
			if err := p.consul.DestroySession(); err != nil {
				panic(err)
			}
		}
		log.Println("\x1b[33;1mStopping Postgres Event Listener:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m")
	}()

	if !p.isHaveJob {
		return
	}

	shutdown <- struct{}{}
	runningJob := len(semaphore)
	if runningJob != 0 {
		fmt.Printf("\x1b[34;1mPostgres Event Listener:\x1b[0m waiting %d job until done...\n", runningJob)
	}

	p.wg.Wait()
}