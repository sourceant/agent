package api

import (
	"context"
	"log"
	"time"

	"github.com/sourceant/agent/internal/core"
)

// A repository read once is a repository that answers about last month. The
// agent is the process that is always up, so it is the one that reads them
// again.
//
// How often is the core's to declare and a person's to set, and it is read
// every time rather than at startup: somebody changing it in Settings should
// not have to restart anything to see it take.

const (
	// How often the settings are consulted. Short enough that a change in
	// Settings takes hold while somebody is still looking at the screen.
	asking = time.Minute

	// Nothing is read again sooner than this, whatever a setting says, because
	// a person who typed 1 by accident should not have their laptop reading
	// every repository they own once a minute.
	sooner = 5 * time.Minute
)

type schedule struct {
	reader Reader
	// When each thing last ran, so a restart does not re-read everything and
	// a long interval is not lost.
	indexed time.Time
	learned time.Time
}

// Keep runs the reading somebody asked for, until ctx is cancelled.
func (s *Server) Keep(ctx context.Context) {
	keeper := &schedule{reader: s.reader}
	ticker := time.NewTicker(asking)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			keeper.tick(ctx)
		}
	}
}

func (k *schedule) tick(ctx context.Context) {
	settings, err := k.reader.Settings(ctx)
	if err != nil {
		// A core that is still starting is not a problem worth logging every
		// minute. The next tick asks again.
		return
	}

	if due(minutes(settings, "index.every"), k.indexed) {
		k.indexed = time.Now()
		k.read(ctx)
	}
	if due(minutes(settings, "knowledge.every"), k.learned) {
		k.learned = time.Now()
		k.learn(ctx, configured(settings))
	}
}

// due reports whether something set to run every so often is owed a run.
// Zero minutes means somebody turned it off.
func due(every time.Duration, last time.Time) bool {
	if every <= 0 {
		return false
	}
	if every < sooner {
		every = sooner
	}
	return time.Since(last) >= every
}

func minutes(settings []core.Setting, key string) time.Duration {
	for _, setting := range settings {
		if setting.Key != key {
			continue
		}
		switch value := setting.Value.(type) {
		case float64:
			return time.Duration(value) * time.Minute
		case int:
			return time.Duration(value) * time.Minute
		}
		return 0
	}
	return 0
}

func (k *schedule) read(ctx context.Context) {
	// Only what changed is read, so this costs close to nothing on a
	// repository nobody has touched.
	if _, err := k.reader.Index(ctx, "", false, true); err != nil {
		log.Printf("reading repositories again: %v", err)
	}
}

// configured reports whether there is a model to ask. Asking core to use one
// it does not have is refused, and a scheduler that did it anyway would put a
// failure in the log every hour for a setting nobody turned on.
func configured(settings []core.Setting) bool {
	named, keyed := false, false
	for _, setting := range settings {
		switch setting.Key {
		case "model.name":
			text, ok := setting.Value.(string)
			named = ok && text != ""
		case "model.api_key":
			keyed = setting.IsSet != nil && *setting.IsSet
		}
	}
	return named && keyed
}

func (k *schedule) learn(ctx context.Context, ask bool) {
	repositories, err := k.reader.Repositories(ctx)
	if err != nil {
		log.Printf("looking for new knowledge: %v", err)
		return
	}
	for _, repository := range repositories {
		// Reading what a repository states costs nothing. Asking a model costs
		// money every time, so it happens only where somebody has said which
		// model to ask.
		if _, err := k.reader.Initialize(ctx, repository.Name, false, ask); err != nil {
			log.Printf("looking for new knowledge in %s: %v", repository.Name, err)
		}
	}
}
