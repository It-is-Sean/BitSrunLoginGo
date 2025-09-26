package login

import (
	"context"
	"fmt"
	"github.com/Mmx233/BackoffCli/backoff"
	"github.com/Mmx233/BitSrunLoginGo/internal/config"
	"github.com/Mmx233/BitSrunLoginGo/internal/config/keys"
	"github.com/Mmx233/BitSrunLoginGo/internal/dns"
	"github.com/Mmx233/BitSrunLoginGo/internal/http_client"
	"github.com/Mmx233/BitSrunLoginGo/internal/webhook"
	"github.com/Mmx233/BitSrunLoginGo/pkg/srun"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// LoginForAccount handles the login process for a single account.
func LoginForAccount(logger log.FieldLogger, account config.Account, eventQueue webhook.EventQueue) error {
	logger = logger.WithField("account", account.Username)
	eventContext := fmt.Sprintf("login_%s", account.Username)

	eventQueue.AddEvent(webhook.NewDataEvent(webhook.ProcessBegin, eventContext, nil))

	loginFunc := func(ctx context.Context) error {
		return doLogin(logger, account, eventQueue, eventContext)
	}

	var err error
	if config.Settings.Backoff.Enable {
		err = backoff.NewInstance(loginFunc, config.BackoffConfig).Run(context.TODO())
	} else {
		err = loginFunc(context.TODO())
	}

	if err != nil {
		logger.Errorln("Login failed after retries: ", err)
		eventQueue.AddEvent(webhook.NewActionFailureEvent(webhook.Login, eventContext, nil, err.Error()))
	} else {
		logger.Infoln("Login successful")
		eventQueue.AddEvent(webhook.NewActionSuccessEvent(webhook.Login, eventContext, nil, account.Username))
	}

	eventQueue.AddEvent(webhook.NewDataEvent(webhook.ProcessFinish, eventContext, nil))
	return err
}

func doLogin(logger log.FieldLogger, account config.Account, eventQueue webhook.EventQueue, eventContext string) error {
	logger.Infoln("Attempting to login...")

	httpClient, err := http_client.NewClient(account.NetIface)
	if err != nil {
		return fmt.Errorf("failed to create http client: %w", err)
	}

	loginInfo := srun.LoginInfo{
		Form: srun.LoginForm{
			Username: account.Username,
			Password: account.Password,
			UserType: account.UserType,
			Domain:   config.Settings.Basic.Domain,
		},
		Meta: *config.Meta,
	}

	srunClient := srun.New(&srun.Conf{
		Logger:       logger,
		Https:        config.Settings.Basic.Https,
		LoginInfo:    loginInfo,
		Client:       httpClient,
		CustomHeader: config.Settings.CustomHeader,
	})

	srunDetector := srunClient.Api.NewDetector()

	// Acid and Enc detection can be added here if needed, following the logic from the old doLogin

	status, ip, err := srunClient.LoginStatus()
	if err != nil {
		// Handle errors, maybe the response is missing IP but still indicates online status
		if status == nil {
			return fmt.Errorf("failed to get login status: %w", err)
		}
		logger.Warnf("Could not get client IP from login status: %v", err)
	}

	var clientIp string
	if ip != nil {
		clientIp = *ip
	}

	if status != nil && *status {
		logger.Infoln("User is already online. IP: ", clientIp)
		if config.Settings.DDNS.Enable {
			_ = ddns(logger, clientIp, httpClient, eventQueue, eventContext)
		}
		return nil
	}

	logger.Infoln("User is offline, proceeding with login.")

	loginIp := ""
	if !config.Meta.DoubleStack {
		if clientIp == "" {
			// If we couldn't get the IP from LoginStatus, try to detect it.
			detectedIp, detectErr := srunDetector.DetectIp()
			if detectErr != nil {
				return fmt.Errorf("failed to detect client IP: %w", detectErr)
			}
			clientIp = detectedIp
		}
		loginIp = clientIp
	}

	if err = srunClient.DoLogin(loginIp); err != nil {
		return err
	}

	logger.Infoln("Login successful. IP: ", clientIp)

	if config.Settings.DDNS.Enable {
		_ = ddns(logger, clientIp, httpClient, eventQueue, eventContext)
	}

	return nil
}

func ddns(logger log.FieldLogger, ip string, httpClient *http.Client, eventQueue webhook.EventQueue, eventContext string) error {
	logger.Infoln("DDNS update triggered.")
	err := dns.Run(&dns.Config{
		Logger:   logger.WithField(keys.LogLoginModule, "ddns"),
		Provider: config.Settings.DDNS.Provider,
		IP:       ip,
		Domain:   config.Settings.DDNS.Domain,
		TTL:      config.Settings.DDNS.TTL,
		Conf:     config.Settings.DDNS.Config,
		Http:     httpClient,
	})

	prop := []webhook.PropertyElement{
		{
			Name:  "domain",
			Value: config.Settings.DDNS.Domain,
		},
	}

	if err != nil {
		eventQueue.AddEvent(webhook.NewActionFailureEvent(webhook.DNSUpdate, eventContext, prop, err.Error()))
	} else {
		eventQueue.AddEvent(webhook.NewActionSuccessEvent(webhook.DNSUpdate, eventContext, prop, ip))
	}
	return err
}