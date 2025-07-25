package service

import (
	"errors"
	"go.uber.org/zap"
	"server/model"
	"server/pkg/jwt"
	"server/pkg/utils"
	"server/repository"
	cache2 "server/repository/cache"
	"server/repository/query"
	"server/service/common/node_rule"
	"server/service/engine"
	"time"
)

type CreateReq struct {
	Name        string `binding:"required" json:"name" label:"名称"`
	TargetIp    string `binding:"required" json:"targetIp" label:"内网IP"`
	TargetPort  string `binding:"required" json:"targetPort" label:"内网端口"`
	TargetHttps int    `json:"targetHttps"`
	NodeCode    string `binding:"required" json:"nodeCode" label:"节点编号"`
	ClientCode  string `binding:"required" json:"clientCode" label:"客户端编号"`
	ConfigCode  string `binding:"required" json:"configCode" label:"套餐配置"`
}

func (service *service) Create(claims jwt.Claims, req CreateReq) error {
	db, _, log := repository.Get("")
	if !utils.ValidateLocalIP(req.TargetIp) {
		return errors.New("内网IP格式错误")
	}
	if !utils.ValidatePort(req.TargetPort) {
		return errors.New("内网端口格式错误")
	}

	var cfg model.SystemConfigGost
	cache2.GetSystemConfigGost(&cfg)
	if cfg.FuncWeb != "1" {
		return errors.New("管理员未启用该功能")
	}

	return db.Transaction(func(tx *query.Query) error {
		user, _ := tx.SystemUser.Where(tx.SystemUser.Code.Eq(claims.Code)).First()
		if user == nil {
			return errors.New("用户错误")
		}

		var domainPrefix = utils.RandStr(8, utils.LatterDict)
		node, _ := tx.GostNode.Where(tx.GostNode.Code.Eq(req.NodeCode)).First()
		if node == nil {
			return errors.New("节点错误")
		}
		if !node.CheckDomainPrefix(domainPrefix) {
			return errors.New("该域名前缀被禁止使用")
		}
		if node.Web != 1 {
			return errors.New("该节点未启用域名解析功能")
		}

		if err := node_rule.VerifyAll(tx, user.Code, node.GetRules()); err != nil {
			return err
		}

		if err := tx.GostNodeDomain.Create(&model.GostNodeDomain{
			Prefix:   domainPrefix,
			NodeCode: node.Code,
		}); err != nil {
			return errors.New("该域名前缀已被使用")
		}
		cfg, _ := tx.GostNodeConfig.Where(
			tx.GostNodeConfig.Code.Eq(req.ConfigCode),
			tx.GostNodeConfig.NodeCode.Eq(node.Code),
		).First()
		if cfg == nil {
			return errors.New("套餐错误")
		}
		client, _ := tx.GostClient.Where(
			tx.GostClient.UserCode.Eq(claims.Code),
			tx.GostClient.Code.Eq(req.ClientCode),
		).First()
		if client == nil {
			return errors.New("客户端错误")
		}

		var expAt = time.Now().Unix()
		switch cfg.ChargingType {
		case model.GOST_CONFIG_CHARGING_CUCLE_DAY:
			expAt = time.Now().Add(time.Duration(cfg.Cycle) * 24 * time.Hour).Unix()
			if user.Amount.LessThan(cfg.Amount) {
				return errors.New("积分不足")
			}

			if _, err := tx.SystemUser.Where(
				tx.SystemUser.Code.Eq(user.Code),
				tx.SystemUser.Version.Eq(user.Version),
			).UpdateSimple(
				tx.SystemUser.Amount.Value(user.Amount.Sub(cfg.Amount)),
				tx.SystemUser.Version.Value(user.Version+1),
			); err != nil {
				log.Error("扣减积分失败", zap.Error(err))
				return errors.New("操作失败")
			}
		case model.GOST_CONFIG_CHARGING_ONLY_ONCE:
			if user.Amount.LessThan(cfg.Amount) {
				return errors.New("积分不足")
			}
			if _, err := tx.SystemUser.Where(
				tx.SystemUser.Code.Eq(user.Code),
				tx.SystemUser.Version.Eq(user.Version),
			).UpdateSimple(
				tx.SystemUser.Amount.Value(user.Amount.Sub(cfg.Amount)),
				tx.SystemUser.Version.Value(user.Version+1),
			); err != nil {
				log.Error("扣减积分失败", zap.Error(err))
				return errors.New("操作失败")
			}
		}

		var host = model.GostClientHost{
			Name:         req.Name,
			TargetIp:     req.TargetIp,
			TargetPort:   req.TargetPort,
			TargetHttps:  req.TargetHttps,
			DomainPrefix: domainPrefix,
			NodeCode:     req.NodeCode,
			ClientCode:   req.ClientCode,
			UserCode:     claims.Code,
			GostClientConfig: model.GostClientConfig{
				ChargingType: cfg.ChargingType,
				Cycle:        cfg.Cycle,
				Amount:       cfg.Amount,
				Limiter:      cfg.Limiter,
				//RLimiter:     cfg.RLimiter,
				//CLimiter:     cfg.CLimiter,
				ExpAt: expAt,
			},
		}
		if err := tx.GostClientHost.Create(&host); err != nil {
			log.Error("新增用户域名解析失败", zap.Error(err))
			return errors.New("操作失败")
		}
		var auth = model.GostAuth{
			TunnelType: model.GOST_TUNNEL_TYPE_HOST,
			TunnelCode: host.Code,
			User:       utils.RandStr(10, utils.AllDict),
			Password:   utils.RandStr(10, utils.AllDict),
		}
		if err := tx.GostAuth.Create(&auth); err != nil {
			log.Error("生成授权信息失败", zap.Error(err))
			return errors.New("操作失败")
		}
		cache2.SetGostAuth(auth.User, auth.Password, host.Code)
		engine.ClientHostConfig(tx, host.Code)
		cache2.SetTunnelInfo(cache2.TunnelInfo{
			Code:        host.Code,
			Type:        model.GOST_TUNNEL_TYPE_HOST,
			ClientCode:  host.ClientCode,
			UserCode:    host.UserCode,
			NodeCode:    host.NodeCode,
			ChargingTye: host.ChargingType,
			ExpAt:       host.ExpAt,
			Limiter:     host.Limiter,
		})
		return nil
	})
}
