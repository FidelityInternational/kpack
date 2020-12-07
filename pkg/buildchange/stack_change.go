package buildchange

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
)

func NewStackChange(oldRunImageRefStr, newRunImageRefStr string) Change {
	var change stackChange
	var errStrs []string

	oldRunImageRef, err := name.ParseReference(oldRunImageRefStr)
	if err != nil {
		errStrs = append(errStrs, err.Error())
	} else {
		change.oldRunImageId = oldRunImageRef.Identifier()
	}

	newRunImageRef, err := name.ParseReference(newRunImageRefStr)
	if err != nil {
		errStrs = append(errStrs, err.Error())
	} else {
		change.newRunImageId = newRunImageRef.Identifier()
	}

	if len(errStrs) > 0 {
		change.err = errors.New(strings.Join(errStrs, "; "))
	}
	return change
}

type stackChange struct {
	oldRunImageId string
	newRunImageId string
	err           error
}

func (s stackChange) Reason() v1alpha1.BuildReason { return v1alpha1.BuildReasonStack }

func (s stackChange) IsValid() (bool, error) {
	return s.oldRunImageId != s.newRunImageId, s.err
}

func (s stackChange) Old() interface{} { return s.oldRunImageId }

func (s stackChange) New() interface{} { return s.newRunImageId }
