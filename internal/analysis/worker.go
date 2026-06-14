package analysis

import (
	"context"
	"log"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const analysisModel = "llm-classifier"

// Worker polls for interactions that have no analysis yet and classifies them.
// It runs in-process (started from cmd/mqtt) and is best-effort: a failing
// interaction is logged and retried on the next tick (it stays "unanalyzed").
type Worker struct {
	pool       *pgxpool.Pool
	classifier *Classifier
	interval   time.Duration
	batchSize  int32
}

func NewWorker(pool *pgxpool.Pool, classifier *Classifier, interval time.Duration, batchSize int32) *Worker {
	return &Worker{pool: pool, classifier: classifier, interval: interval, batchSize: batchSize}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	log.Printf("[analysis] worker started (interval=%s batch=%d)", w.interval, w.batchSize)
	for {
		select {
		case <-ctx.Done():
			log.Println("[analysis] worker stopped")
			return
		case <-t.C:
			if n, err := w.processBatch(ctx); err != nil {
				log.Printf("[analysis] batch error: %v", err)
			} else if n > 0 {
				log.Printf("[analysis] analyzed %d interactions", n)
			}
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) (int, error) {
	pending, err := store.New(w.pool).ListUnanalyzedInteractions(ctx, w.batchSize)
	if err != nil {
		return 0, err
	}
	done := 0
	for _, it := range pending {
		if err := w.analyzeOne(ctx, it); err != nil {
			log.Printf("[analysis] interaction %s: %v", it.ID, err)
			continue
		}
		done++
	}
	return done, nil
}

// analyzeOne classifies a single interaction and writes the analysis, its topics
// and the per-child topic stats atomically. If anything fails the tx rolls back
// and the interaction is picked up again next tick.
func (w *Worker) analyzeOne(ctx context.Context, it store.Interaction) error {
	cl, err := w.classifier.Classify(ctx, it.Transcript)
	if err != nil {
		return err
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := store.New(tx)

	if _, err := q.UpsertInteractionAnalysis(ctx, store.UpsertInteractionAnalysisParams{
		InteractionID:   it.ID,
		CognitiveArea:   cl.CognitiveArea,
		QuestionType:    cl.QuestionType,
		ComplexityLevel: cl.Complexity,
		Sentiment:       cl.Sentiment,
		Model:           analysisModel,
	}); err != nil {
		return err
	}

	for _, tp := range cl.Topics {
		if tp.Slug == "" {
			continue
		}
		topic, err := q.UpsertTopic(ctx, store.UpsertTopicParams{
			Slug:  tp.Slug,
			Label: tp.Label,
			Area:  cl.CognitiveArea,
		})
		if err != nil {
			return err
		}
		if err := q.LinkInteractionTopic(ctx, store.LinkInteractionTopicParams{
			InteractionID: it.ID,
			TopicID:       topic.ID,
		}); err != nil {
			return err
		}
		// child_topic_stats powers the interest graph; only meaningful when the
		// interaction was attributed to a child.
		if it.ChildID.Valid {
			if err := q.BumpChildTopicStat(ctx, store.BumpChildTopicStatParams{
				ChildID: uuid.UUID(it.ChildID.Bytes),
				TopicID: topic.ID,
			}); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}
