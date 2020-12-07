package buildchange

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
)

const (
	reasonsSeparator = ","
	errorSeparator = "\n"
)

func NewChangeProcessor() *ChangeProcessor {
	return &ChangeProcessor{
		changes: []GenericChange{},
		errStrs: []string{},
	}
}

type ChangeProcessor struct {
	changes []GenericChange
	errStrs []string
}

func (c *ChangeProcessor) Process(change Change) *ChangeProcessor {
	if change == nil {
		return c
	}

	if !change.Reason().IsValid() {
		errStr := fmt.Sprintf("unsupported change reason '%s'", change.Reason())
		c.errStrs = append(c.errStrs, errStr)
		return c
	}

	valid, err := change.IsValid()
	if err != nil {
		err := errors.Wrapf(err, "error validating change for reason '%s'", change.Reason())
		c.errStrs = append(c.errStrs, err.Error())
	} else if valid {
		c.changes = append(c.changes, newGenericChange(change))
	}

	return c
}

func (c *ChangeProcessor) Summarize() (ChangeSummary, error) {
	changesStr, err := c.changesStr()
	if err != nil {
		err := errors.Wrapf(err, "error generating changes string")
		c.errStrs = append(c.errStrs, err.Error())
	}

	summary, err := NewChangeSummary(c.hasChanges(), c.reasonsStr(), changesStr)
	if err != nil {
		err := errors.Wrapf(err, "error summarizing changes")
		c.errStrs = append(c.errStrs, err.Error())
	}

	if len(c.errStrs) > 0 {
		return summary, errors.New(strings.Join(c.errStrs, errorSeparator))
	}

	return summary, nil
}

func (c *ChangeProcessor) hasChanges() bool {
	return len(c.changes) > 0
}

func (c *ChangeProcessor) reasonsStr() string {
	if !c.hasChanges() {
		return ""
	}

	var reasons = make([]string, len(c.changes))
	for i, change := range c.changes {
		reasons[i] = change.Reason
	}

	sort.SliceStable(reasons, func(i, j int) bool {
		return strings.Index(v1alpha1.BuildReasonSortIndex, reasons[i]) <
			strings.Index(v1alpha1.BuildReasonSortIndex, reasons[j])
	})

	return strings.Join(reasons, reasonsSeparator)
}

func (c *ChangeProcessor) changesStr() (string, error) {
	if !c.hasChanges() {
		return "", nil
	}

	sort.SliceStable(c.changes, func(i, j int) bool {
		return strings.Index(v1alpha1.BuildReasonSortIndex, c.changes[i].Reason) <
			strings.Index(v1alpha1.BuildReasonSortIndex, c.changes[j].Reason)
	})

	bytes, err := json.Marshal(c.changes)
	if err != nil {
		return "", err
	}

	return string(bytes), err
}
