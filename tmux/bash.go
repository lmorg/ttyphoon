package tmux

import (
	_ "embed"
	"net/url"
	"os"
	"text/template"
)

//go:embed bash.sh.tmpl
var bashScriptTemplate string

var bashScript = template.Must(template.New("bash.sh.tmpl").Funcs(template.FuncMap{
	"shellQuote": quoteShellArgument,
}).Parse(bashScriptTemplate))

func writeBashScript(commandLine, callbackID, pwd string) (string, error) {
	file, err := os.CreateTemp("", "ttyphoon-*.sh")
	if err != nil {
		return "", err
	}
	err = bashScript.Execute(file, struct {
		CommandLine string
		CallbackID  string
		PathURI     string
	}{commandLine, callbackID, (&url.URL{Path: pwd}).EscapedPath()})
	_ = file.Close()
	return file.Name(), err
}
