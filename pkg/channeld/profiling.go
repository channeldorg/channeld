package channeld

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/profile"
)

func StartProfiling() {
	if GlobalSettings.ProfileOption != nil {
		prof := profile.Start(
			GlobalSettings.ProfileOption,
			profile.ProfilePath(GlobalSettings.ProfilePath),
			profile.NoShutdownHook,
		)
		// Replace pprof's default shutdown hook with more signals to make sure the file is saved before the process is terminated.
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
			<-c

			log.Println("profile: caught interrupt, stopping profiles")
			prof.Stop()

			os.Exit(0)
		}()
	}
}
