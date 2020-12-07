package buildchange

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
)

func NewConfigChange(oldConfig, newConfig Config) Change {
	return configChange{
		old: oldConfig,
		new: newConfig,
	}
}

type configChange struct {
	old Config
	new Config
}

type Config struct {
	Env       []corev1.EnvVar             `json:"env,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	Bindings  v1alpha1.Bindings           `json:"bindings,omitempty"`
	Source    v1alpha1.SourceConfig       `json:"source,omitempty"`
}

func (c configChange) Reason() v1alpha1.BuildReason { return v1alpha1.BuildReasonConfig }

func (c configChange) IsValid() (bool, error) {
	// Git revision changes are considered as COMMIT change
	// Ignore them as part of CONFIG Change
	var oldGitRevision, newGitRevision string

	if c.old.Source.Git != nil {
		oldGitRevision = c.old.Source.Git.Revision
		c.old.Source.Git.Revision = ""
	}
	if c.new.Source.Git != nil {
		newGitRevision = c.new.Source.Git.Revision
		c.new.Source.Git.Revision = ""
	}

	valid := !equality.Semantic.DeepEqual(c.old, c.new)

	if c.old.Source.Git != nil {
		c.old.Source.Git.Revision = oldGitRevision
	}
	if c.new.Source.Git != nil {
		c.new.Source.Git.Revision = newGitRevision
	}
	return valid, nil
}

func (c configChange) Old() interface{} { return c.old }

func (c configChange) New() interface{} { return c.new }
