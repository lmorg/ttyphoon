package dispatcher

var cleanUpFuncs []func()

func CleanUp() {
	for _, fn := range cleanUpFuncs {
		fn()
	}
}
