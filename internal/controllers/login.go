package controllers

import (
	"errors"
	"github.com/Mmx233/BitSrunLoginGo/internal/config"
	"github.com/Mmx233/BitSrunLoginGo/internal/config/flags"
	"github.com/Mmx233/BitSrunLoginGo/internal/pkg/dns"
	"github.com/Mmx233/BitSrunLoginGo/internal/pkg/http_client"
	"github.com/Mmx233/BitSrunLoginGo/pkg/srun"
	"github.com/Mmx233/BitSrunLoginGo/tools"
	"net/http"
)

// Login 登录逻辑
func Login(eth *tools.Eth, debugOutput bool) error {
	logger := config.Logger

	// 登录配置初始化
	httpClient := http_client.HttpPackSelect(eth).Client
	srunClient := srun.New(&srun.Conf{
		Logger: logger,
		Https:  config.Settings.Basic.Https,
		LoginInfo: srun.LoginInfo{
			Form: *config.Form,
			Meta: *config.Meta,
		},
		Client:       httpClient,
		CustomHeader: config.Settings.CustomHeader,
	})

	srunDetector := srunClient.Api.NewDetector()

	// Reality 与 Acid
	var acidOnReality bool
	if config.Settings.Reality.Enable {
		logger.Debugln("开始 Reality 流程")
		acid, _, err := srunDetector.Reality(config.Settings.Reality.Addr, flags.AutoAcid)
		if err != nil {
			logger.Warnln("Reality 请求异常:", err)
		} else if flags.AutoAcid && acid != "" {
			acidOnReality = true
			logger.Debugf("使用嗅探 acid: %s", acid)
			srunClient.LoginInfo.Meta.Acid = acid
		}
	}
	if !acidOnReality && flags.AutoAcid {
		logger.Debugln("开始嗅探 acid")
		acid, err := srunDetector.DetectAcid()
		if err != nil {
			if errors.Is(err, srun.ErrAcidCannotFound) {
				logger.Warnln("找不到 acid，使用配置 acid")
			} else {
				logger.Warnf("嗅探 acid 失败，使用配置 acid: %v", err)
			}
		} else {
			logger.Debugf("使用嗅探 acid: %s", acid)
			srunClient.LoginInfo.Meta.Acid = acid
		}
	}

	if flags.AutoEnc {
		logger.Debugln("开始嗅探 enc")
		enc, err := srunDetector.DetectEnc()
		if err != nil {
			if errors.Is(err, srun.ErrEnvCannotFound) {
				logger.Warnln("找不到 enc，使用配置 enc")
			} else {
				logger.Warnf("嗅探 enc 失败，使用配置 enc: %v", err)
			}
		} else {
			logger.Debugf("使用嗅探 enc: %s", enc)
			srunClient.LoginInfo.Meta.Enc = enc
		}
	}

	// 选择输出函数
	var output func(args ...interface{})
	if debugOutput {
		output = logger.Debugln
	} else {
		output = logger.Infoln
	}

	output("正在获取登录状态")

	online, ip, err := srunClient.LoginStatus()
	if err != nil {
		return err
	}

	var loginIp string

	if config.Meta.DoubleStack {
		logger.Debugln("使用双栈网络时认证 ip 为空")
	} else {
		loginIp = ip
		logger.Debugln("认证客户端 ip: ", ip)
	}

	// 登录执行

	if online {
		output("已登录~")

		if config.Settings.DDNS.Enable && config.Settings.Guardian.Enable && ipLast != ip {
			if ddns(ip, httpClient) == nil {
				ipLast = ip
			}
		}

		return nil
	} else {
		logger.Infoln("检测到用户未登录，开始尝试登录...")

		if err = srunClient.DoLogin(loginIp); err != nil {
			return err
		}

		logger.Infoln("登录成功~")

		if config.Settings.DDNS.Enable {
			_ = ddns(ip, httpClient)
		}
	}

	return nil
}

var ipLast string

func ddns(ip string, httpClient *http.Client) error {
	return dns.Run(&dns.Config{
		Logger:   config.Logger,
		Provider: config.Settings.DDNS.Provider,
		IP:       ip,
		Domain:   config.Settings.DDNS.Domain,
		TTL:      config.Settings.DDNS.TTL,
		Conf:     config.Settings.DDNS.Config,
		Http:     httpClient,
	})
}
