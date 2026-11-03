package agent

import (
	"context"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

type skillResourceDirectoriesContextKey struct{}

func WithSkillResourceDirectories(ctx context.Context, directories map[string]string) context.Context {
	if ctx == nil || len(directories) == 0 {
		return ctx
	}
	snapshot := make(map[string]string, len(directories))
	for name, directory := range directories {
		snapshot[name] = directory
	}
	return context.WithValue(ctx, skillResourceDirectoriesContextKey{}, snapshot)
}

func SkillResourceDirectoriesFromContext(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}
	directories, _ := ctx.Value(skillResourceDirectoriesContextKey{}).(map[string]string)
	return directories
}

func (agt *Agent) SetSkillResourceDirectories(directories map[string]string) {
	if agt.Meta == nil {
		agt.Meta = &aitypes.Meta{}
	}
	if sameSkillResourceDirectories(agt.Meta.SkillResources, directories) {
		return
	}
	if len(directories) == 0 {
		agt.Meta.SkillResources = nil
	} else {
		snapshot := make(map[string]string, len(directories))
		for name, directory := range directories {
			snapshot[name] = directory
		}
		agt.Meta.SkillResources = snapshot
	}
	agt.Reload()
}

func sameSkillResourceDirectories(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for name, directory := range left {
		if right[name] != directory {
			return false
		}
	}
	return true
}
