package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"bitbucket.org/ipowow/updater"
)

func main() {
	// Instantiates the CoreRoller updater to check periodically for version update.
	if updater, err := updater.New(30*time.Second, syscall.SIGTERM); err == nil {
		go updater.Start()
	}

	// Your app code
	// ...

	// Wait for signal to terminate
	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh, os.Interrupt, syscall.SIGTERM)
	<-signalsCh
}
