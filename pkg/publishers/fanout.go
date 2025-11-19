package publishers

import (
	"context"
	"errors"
	"fmt"
)

// Fanout dispatches events to all configured publishers.
type Fanout struct {
	publishers []Publisher
}

// NewFanout builds a dispatcher that fans out events across publishers.
func NewFanout(pubs []Publisher) *Fanout {
	cp := make([]Publisher, 0, len(pubs))
	for _, p := range pubs {
		if p == nil {
			continue
		}
		cp = append(cp, p)
	}
	return &Fanout{publishers: cp}
}

// Publish forwards the event to every registered publisher.
func (f *Fanout) Publish(ctx context.Context, evt Event) error {
	if f == nil || len(f.publishers) == 0 {
		return nil
	}

	var errs []error
	for _, p := range f.publishers {
		if err := p.Publish(ctx, evt); err != nil {
			errs = append(errs, fmt.Errorf("%s publisher[%s]: %w", p.Type(), p.ID(), err))
		}
	}
	return errors.Join(errs...)
}

// Size returns the number of active publishers.
func (f *Fanout) Size() int {
	if f == nil {
		return 0
	}
	return len(f.publishers)
}
