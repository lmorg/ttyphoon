package prompts

import (
	"fmt"
	"runtime"

	"github.com/lmorg/mxtty/ai/agent"

	_ "embed"
)

var (
	//go:embed common.md
	mdCommon string

	//go:embed ask.md
	mdAsk string

	//go:embed explain.md
	mdExplain string
)

func GetExplain(meta *agent.Meta, userPrompt string) string {
	return fmt.Sprintf(
		"%s\nOperating system: %s, CPU: %s.\n%s\n%s\nCommand line executed: %s\nCommand line output below:\n%s",
		mdCommon+mdExplain, runtime.GOOS, runtime.GOARCH, meta.History.String(), userPrompt, meta.CmdLine, meta.OutputBlock)
}

func GetAsk(meta *agent.Meta, userPrompt string) string {
	return fmt.Sprintf(
		"%sOperating system: %s, CPU: %s.\n%s\n%s",
		mdCommon+mdAsk, runtime.GOOS, runtime.GOARCH, meta.History.String(), userPrompt)
}
