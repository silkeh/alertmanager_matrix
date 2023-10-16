package alertmanager

import (
	"time"

	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/pkg/labels"

	"gitlab.com/slxh/matrix/alertmanager_matrix/internal/util"
)

// Silence represents a silence configured in Alertmanager.
type Silence struct {
	*models.GettableSilence
}

// ID returns the ID of the Silence.
func (s *Silence) ID() string {
	return util.ValueOrDefault(s.GettableSilence.ID)
}

// Status returns the status state string of the Silence.
func (s *Silence) Status() string {
	return util.ValueOrDefault(util.ValueOrDefault(s.GettableSilence.Status).State)
}

// Comment returns the comment associated with the Silence.
func (s *Silence) Comment() string {
	return util.ValueOrDefault(s.GettableSilence.Comment)
}

// CreatedBy returns the creator of the Silence.
func (s *Silence) CreatedBy() string {
	return util.ValueOrDefault(s.GettableSilence.CreatedBy)
}

// Matchers returns the [label.Matchers] for the Silence.
func (s *Silence) Matchers() labels.Matchers {
	return decodeMatchers(s.GettableSilence.Matchers)
}

// SetMatchers sets the Matchers based on the given [label.Matchers].
func (s *Silence) SetMatchers(matchers labels.Matchers) {
	s.GettableSilence.Matchers = encodeMatchers(matchers)
}

// StartsAt returns the time that the Silence starts at.
func (s *Silence) StartsAt() time.Time {
	return time.Time(util.ValueOrDefault(s.GettableSilence.EndsAt))
}

// EndsAt returns the time that the Silence ends at.
func (s *Silence) EndsAt() time.Time {
	return time.Time(util.ValueOrDefault(s.GettableSilence.EndsAt))
}

// UpdatedAt returns the time when the Silence was last updated.
func (s *Silence) UpdatedAt() time.Time {
	return time.Time(util.ValueOrDefault(s.GettableSilence.UpdatedAt))
}

func encodeMatchers(matchers labels.Matchers) models.Matchers {
	ms := make(models.Matchers, len(matchers))

	for i, m := range matchers {
		ms[i] = &models.Matcher{
			IsEqual: util.PtrTo(m.Type == labels.MatchEqual || m.Type == labels.MatchRegexp),
			IsRegex: util.PtrTo(m.Type == labels.MatchRegexp || m.Type == labels.MatchNotRegexp),
			Name:    util.PtrTo(m.Name),
			Value:   util.PtrTo(m.Value),
		}
	}

	return ms
}

func decodeMatchers(matchers models.Matchers) labels.Matchers {
	ms := make(labels.Matchers, len(matchers))

	for i, m := range matchers {
		ms[i] = &labels.Matcher{
			Type:  matcherType(util.ValueOrDefault(m.IsEqual), util.ValueOrDefault(m.IsRegex)),
			Name:  util.ValueOrDefault(m.Name),
			Value: util.ValueOrDefault(m.Value),
		}
	}

	return ms
}

func matcherType(isEqual, isRegex bool) labels.MatchType {
	switch {
	case isRegex && isEqual:
		return labels.MatchRegexp
	case isRegex && !isEqual:
		return labels.MatchNotRegexp
	case isEqual:
		return labels.MatchEqual
	default:
		return labels.MatchNotEqual
	}
}
