package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bitbucket.org/ipowow/updater"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mitchellh/goamz/aws"
)

var config struct {
	etcdHost     string
	etcdPath     string
	awsAccessKey string
	awsSecretKey string
}

func init() {
	flag.StringVar(&config.etcdHost, "etcd-host", "http://localhost:2379", "Etcd service address")
	flag.StringVar(&config.etcdPath, "config-path", "/lbManager", "Configuration path")
	flag.StringVar(&config.awsAccessKey, "aws-access-key", "", "AWS access key")
	flag.StringVar(&config.awsSecretKey, "aws-secret-key", "", "AWS secret key")
}

func main() {
	flag.Parse()

	// Instantiates the CoreRoller updater to check periodically for version update.
	if updater, err := updater.New(30*time.Second, syscall.SIGTERM); err == nil {
		go updater.Start()
	}

	awsAuth, err := aws.GetAuth(config.awsAccessKey, config.awsSecretKey)
	if err != nil {
		log.Println(err)
	}

	manager := &Manager{
		configPath: config.etcdPath,
		etcdClient: etcd.NewClient(strings.Split(config.etcdHost, ",")),
		awsAuth:    awsAuth,
	}

	log.Println("Running load balancers manager...")
	go manager.Start()

	// Wait for signal to terminate
	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh, os.Interrupt, syscall.SIGTERM)
	<-signalsCh
}
