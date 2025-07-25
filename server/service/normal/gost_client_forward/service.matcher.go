package service

import (
	"errors"
	"go.uber.org/zap"
	"server/pkg/jwt"
	"server/repository"
	"server/repository/query"
	"server/service/engine"
)

type MatcherReq struct {
	Code       string        `binding:"required" json:"code"`
	Enable     int           `json:"enable"`
	Matchers   []MatcherItem `json:"matchers"`
	TcpMatcher ItemMatcher   `json:"tcpMatcher"`
	SSHMatcher ItemMatcher   `json:"sshMatcher"`
}

type MatcherItem struct {
	Host       string `json:"host"`
	TargetIp   string `json:"targetIp"`
	TargetPort string `json:"targetPort"`
}

func (service *service) Matcher(claims jwt.Claims, req MatcherReq) error {
	db, _, log := repository.Get("")
	return db.Transaction(func(tx *query.Query) error {
		user, _ := tx.SystemUser.Where(tx.SystemUser.Code.Eq(claims.Code)).First()
		if user == nil {
			return errors.New("用户错误")
		}

		forward, _ := tx.GostClientForward.Where(
			tx.GostClientForward.UserCode.Eq(user.Code),
			tx.GostClientForward.Code.Eq(req.Code),
		).First()
		if forward == nil {
			return errors.New("操作失败")
		}

		//forward.MatcherEnable = req.Enable
		//var matchers []model.ForwardMatcher
		//for _, matcher := range req.Matchers {
		//	matchers = append(matchers, model.ForwardMatcher{
		//		Host:       matcher.Host,
		//		TargetIp:   matcher.TargetIp,
		//		TargetPort: matcher.TargetPort,
		//	})
		//}
		//forward.SetMatcher(matchers)
		//forward.SetTcpMatcher(req.TcpMatcher.TargetIp, req.TcpMatcher.TargetPort)
		//forward.SetSSHMatcher(req.SSHMatcher.TargetIp, req.SSHMatcher.TargetPort)

		if err := tx.GostClientForward.Save(forward); err != nil {
			log.Error("修改端口转发失败", zap.Error(err))
			return errors.New("操作失败")
		}
		engine.ClientForwardConfig(tx, forward.Code)
		return nil
	})
}
