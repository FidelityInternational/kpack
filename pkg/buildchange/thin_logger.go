package buildchange

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/pkg/errors"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	"github.com/pivotal/kpack/pkg/differ"
)

func Log(logger *log.Logger, reasonsStr, changesStr string) error {
	return NewThinLogger(logger, reasonsStr, changesStr).Log()
}

type GenericChange struct {
	Old interface{}
	New interface{}
}

type ThinLogger struct {
	logger     *log.Logger
	reasonsStr string
	changesStr string

	differ     differ.Differ
	reasons    []string
	changesMap map[string]GenericChange
}

func NewThinLogger(logger *log.Logger, reasonsStr, changesStr string) *ThinLogger {
	options := differ.DefaultOptions()
	options.Prefix = "\t"

	return &ThinLogger{
		logger:     logger,
		reasonsStr: reasonsStr,
		changesStr: changesStr,
		differ:     differ.NewDiffer(options),
	}
}

func (c *ThinLogger) Log() error {
	if err := c.validate(); err != nil {
		return errors.Wrapf(err, "error validating")
	}

	if err := c.parseReasons(); err != nil {
		return errors.Wrapf(err, "error parsing build reasons string '%s'", c.reasonsStr)
	}

	if err := c.parseChanges(); err != nil {
		return errors.Wrapf(err, "error parsing build changes JSON string '%s'", c.changesStr)
	}

	c.logReasons()
	return c.logChanges()
}

func (c *ThinLogger) validate() error {
	if c.reasonsStr == "" {
		return errors.New("build reasons is empty")
	}
	if c.changesStr == "" {
		return errors.New("build changes is empty")
	}
	return nil
}

func (c *ThinLogger) parseReasons() error {
	c.reasons = strings.Split(c.reasonsStr, reasonsSeparator)
	if len(c.reasons) < 1 {
		return errors.Errorf("error parsing reasons")
	}
	return nil
}

func (c *ThinLogger) parseChanges() (err error) {
	c.changesMap = map[string]GenericChange{}
	if err := json.Unmarshal([]byte(c.changesStr), &c.changesMap); err != nil {
		return err
	}
	return nil
}

func (c *ThinLogger) logReasons() {
	c.logger.Printf("Build reason(s): %s\n", c.reasonsStr)
}

func (c *ThinLogger) logChanges() error {
	for _, reason := range c.reasons {
		change, ok := c.changesMap[reason]
		if !ok {
			return errors.Errorf("changes not available for the reason '%s'", reason)
		}

		if err := c.logChange(reason, change); err != nil {
			return errors.Errorf("error logging change for the reason '%s'", reason)
		}
	}
	return nil
}

func (c *ThinLogger) logChange(reason string, change GenericChange) error {
	if reason == v1alpha1.BuildReasonTrigger {
		c.logger.Print(fmt.Sprintf("%s: %s\n", reason, change.New))
		return nil
	}

	diff, err := c.differ.Diff(change.Old, change.New)
	if err != nil {
		return err
	}

	changeHeader := fmt.Sprintf("%s change:\n", reason)
	c.logger.Printf(changeHeader)
	c.logger.Print(diff)
	return nil
}
