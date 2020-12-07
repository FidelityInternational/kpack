package image

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	"github.com/pivotal/kpack/pkg/buildchange"
)

func NewBuildDeterminer(
	img *v1alpha1.Image,
	lastBuild *v1alpha1.Build,
	srcResolver *v1alpha1.SourceResolver,
	builder v1alpha1.BuilderResource) *BuildDeterminer {

	return &BuildDeterminer{
		img:         img,
		lastBuild:   lastBuild,
		srcResolver: srcResolver,
		builder:     builder,
	}
}

type BuildDeterminer struct {
	img         *v1alpha1.Image
	lastBuild   *v1alpha1.Build
	srcResolver *v1alpha1.SourceResolver
	builder     v1alpha1.BuilderResource

	changeSummary buildchange.ChangeSummary
}

func (b *BuildDeterminer) IsBuildNeeded() (corev1.ConditionStatus, error) {
	if !b.canDetermine() {
		return corev1.ConditionUnknown, nil
	}

	var err error
	b.changeSummary, err = buildchange.NewChangeProcessor().
		Process(b.triggerChange()).
		Process(b.commitChange()).
		Process(b.configChange()).
		Process(b.buildpackChange()).
		Process(b.stackChange()).Summarize()
	if err != nil {
		return corev1.ConditionUnknown, err
	}

	if b.changeSummary.HasChanges {
		return corev1.ConditionTrue, nil
	}
	return corev1.ConditionFalse, nil
}

func (b *BuildDeterminer) Reasons() string {
	return b.changeSummary.ReasonsStr
}

func (b *BuildDeterminer) Changes() string {
	return b.changeSummary.ChangesStr
}

func (b *BuildDeterminer) canDetermine() bool {
	return b.srcResolver.Ready() && b.builder.Ready()
}

func (b *BuildDeterminer) lastBuildWasSuccessful() bool {
	return b.lastBuild != nil && b.lastBuild.IsSuccess()
}

func (b *BuildDeterminer) triggerChange() buildchange.Change {
	if b.lastBuild == nil || b.lastBuild.Annotations == nil {
		return nil
	}

	time, ok := b.lastBuild.Annotations[v1alpha1.BuildNeededAnnotation]
	if !ok {
		return nil
	}

	return buildchange.NewTriggerChange(time)
}

func (b *BuildDeterminer) commitChange() buildchange.Change {
	if b.lastBuild == nil || b.srcResolver.Status.Source.Git == nil {
		return nil
	}

	oldRevision := b.lastBuild.Spec.Source.Git.Revision
	newRevision := b.srcResolver.Status.Source.Git.Revision
	return buildchange.NewCommitChange(oldRevision, newRevision)
}

func (b *BuildDeterminer) configChange() buildchange.Change {
	var old buildchange.Config
	var new buildchange.Config

	if b.lastBuild != nil {
		old = buildchange.Config{
			Env:       b.lastBuild.Spec.Env,
			Resources: b.lastBuild.Spec.Resources,
			Bindings:  b.lastBuild.Spec.Bindings,
			Source:    b.lastBuild.Spec.Source,
		}
	}

	new = buildchange.Config{
		Env:       b.img.Env(),
		Resources: b.img.Resources(),
		Bindings:  b.img.Bindings(),
		Source:    b.srcResolver.Status.Source.ResolvedSource().SourceConfig(),
	}

	return buildchange.NewConfigChange(old, new)
}

func (b *BuildDeterminer) buildpackChange() buildchange.Change {
	if !b.lastBuildWasSuccessful() {
		return nil
	}

	builderBuildpacks := b.builder.BuildpackMetadata()
	getBuilderBuildpackById := func(bpId string) *v1alpha1.BuildpackMetadata {
		for _, bp := range builderBuildpacks {
			if bp.Id == bpId {
				return &bp
			}
		}
		return nil
	}

	var old []v1alpha1.BuildpackInfo
	var new []v1alpha1.BuildpackInfo

	for _, lastBuildBp := range b.lastBuild.Status.BuildMetadata {
		builderBp := getBuilderBuildpackById(lastBuildBp.Id)
		if builderBp == nil {
			old = append(old, v1alpha1.BuildpackInfo{Id: lastBuildBp.Id, Version: lastBuildBp.Version})
		} else if builderBp.Version != lastBuildBp.Version {
			old = append(old, v1alpha1.BuildpackInfo{Id: lastBuildBp.Id, Version: lastBuildBp.Version})
			new = append(new, v1alpha1.BuildpackInfo{Id: builderBp.Id, Version: builderBp.Version})
		}
	}

	return buildchange.NewBuildpackChange(old, new)
}

func (b *BuildDeterminer) stackChange() buildchange.Change {
	if !b.lastBuildWasSuccessful() {
		return nil
	}

	oldRunImageRefStr := b.lastBuild.Status.Stack.RunImage
	newRunImageRefStr := b.builder.RunImage()
	return buildchange.NewStackChange(oldRunImageRefStr, newRunImageRefStr)
}
