package login

import (
	"github.com/Mmx233/BitSrunLoginGo/internal/config"
	"github.com/Mmx233/BitSrunLoginGo/internal/webhook"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Guardian starts a concurrent login-and-keep-alive process for each account in the configuration.
func Guardian(logger log.FieldLogger, eventQueue webhook.EventQueue) {
	var wg sync.WaitGroup

	if len(config.Accounts) == 0 {
		logger.Warnln("No accounts configured. Guardian will not start.")
		return
	}

	for _, account := range config.Accounts {
		wg.Add(1)
		go func(acc config.Account) {
			defer wg.Done()
			guardianForAccount(logger, acc, eventQueue)
		}(account)
	}

	wg.Wait()
}

func guardianForAccount(logger log.FieldLogger, account config.Account, eventQueue webhook.EventQueue) {
	guardianDuration := time.Duration(config.Settings.Guardian.Duration) * time.Second
	accountLogger := logger.WithField("username", account.Username)

	for {
		accountLogger.Infof("Starting login process for account: %s", account.Username)
		if err := LoginForAccount(accountLogger, account, eventQueue); err != nil {
			accountLogger.Errorf("Login process for account %s failed: %v", account.Username, err)
		}
		accountLogger.Infof("Next login attempt for account %s in %v", account.Username, guardianDuration)
		time.Sleep(guardianDuration)
	}
}