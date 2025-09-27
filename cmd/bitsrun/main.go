package main

import (
	"context"
	"github.com/Mmx233/BitSrunLoginGo/internal/config"
	"github.com/Mmx233/BitSrunLoginGo/internal/config/keys"
	"github.com/Mmx233/BitSrunLoginGo/internal/http_client"
	"github.com/Mmx233/BitSrunLoginGo/internal/login"
	"github.com/Mmx233/BitSrunLoginGo/internal/webhook"
	"time"
)

func main() {
	logger := config.Logger

	// Create a new HTTP client for the webhook
	webhookClient, err := http_client.NewClient("")
	if err != nil {
		logger.Fatalf("Failed to create webhook http client: %v", err)
	}

	var _webhook webhook.Webhook
	if config.Settings.Webhook.Enable {
		_webhook = webhook.PostWebhook{
			Url:     config.Settings.Webhook.Url,
			Timeout: time.Duration(config.Settings.Webhook.Timeout) * time.Second,
			Client:  webhookClient,
			Logger:  logger.WithField(keys.LogComponent, "webhook"),
		}
	} else {
		_webhook = webhook.NopWebhook{}
	}
	eventQueue := webhook.NewEventQueue(logger.WithField(keys.LogComponent, "eventQueue"), _webhook)

	if config.Settings.Guardian.Enable {
		logger.Infoln("Guardian mode enabled. Starting continuous login process.")
		login.Guardian(logger.WithField(keys.LogComponent, "guard"), eventQueue)
	} else {
		logger.Infoln("Performing a single login for all configured accounts.")
		for _, account := range config.Accounts {
			if err := login.LoginForAccount(logger, account, eventQueue); err != nil {
				logger.Errorf("Login failed for account %s: %v", account.Username, err)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Settings.Webhook.Timeout)*time.Second)
	defer cancel()
	if err := eventQueue.Close(ctx); err != nil {
		logger.Errorf("Event queue ended with error: %v", err)
	}

	logger.Infoln("Process finished.")
}