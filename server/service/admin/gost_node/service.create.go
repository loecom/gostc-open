package service

import (
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"server/model"
	"server/repository"
	cache2 "server/repository/cache"
	"server/service/common/node_port"
	"strings"
)

type CreateReq struct {
	Name              string   `binding:"required" json:"name" label:"名称"`
	Remark            string   `json:"remark"`
	Rules             []string `json:"rules"`
	Tags              []string `json:"tags"`
	Web               int      `binding:"required" json:"web" label:"是否启用域名解析"`
	Tunnel            int      `binding:"required" json:"tunnel" label:"是否启用私有隧道"`
	Forward           int      `binding:"required" json:"forward" label:"是否启用端口转发"`
	Proxy             int      `binding:"required" json:"proxy" label:"是否启用代理隧道"`
	P2P               int      `binding:"required" json:"p2p" label:"是否启用P2P隧道"`
	Address           string   `binding:"required" json:"address"`
	ReplaceAddress    string   `json:"replaceAddress"`
	Protocol          string   `binding:"required" json:"protocol"`
	HttpPort          string   `json:"httpPort"`
	Domain            string   `json:"domain"`
	DenyDomainPrefix  string   `json:"denyDomainPrefix"`
	UrlTpl            string   `json:"urlTpl"`
	ForwardPorts      string   `json:"forwardPorts"`
	P2PDisableForward int      `json:"p2pDisableForward"`
	IndexValue        int      `json:"indexValue"`

	LimitResetIndex int `json:"limitResetIndex"`
	LimitTotal      int `json:"limitTotal"`
	LimitKind       int `json:"limitKind"`
}

func (service *service) Create(req CreateReq) error {
	db, _, log := repository.Get("")
	var node = model.GostNode{
		Key:               uuid.NewString(),
		Name:              req.Name,
		Remark:            req.Remark,
		Web:               req.Web,
		Tunnel:            req.Tunnel,
		Forward:           req.Forward,
		Proxy:             req.Proxy,
		P2P:               req.P2P,
		Domain:            req.Domain,
		DenyDomainPrefix:  req.DenyDomainPrefix,
		Address:           req.Address,
		ReplaceAddress:    req.ReplaceAddress,
		UrlTpl:            req.UrlTpl,
		Protocol:          req.Protocol,
		HttpPort:          req.HttpPort,
		ForwardPorts:      req.ForwardPorts,
		P2PDisableForward: req.P2PDisableForward,
		Rules:             strings.Join(req.Rules, ","),
		Tags:              strings.Join(req.Tags, ","),
		IndexValue:        req.IndexValue,
		LimitResetIndex:   req.LimitResetIndex,
		LimitTotal:        req.LimitTotal,
		LimitKind:         req.LimitKind,
	}
	if err := db.GostNode.Create(&node); err != nil {
		log.Error("新增节点失败", zap.Error(err))
		return errors.New("操作失败")
	}
	cache2.RefreshNodeObsLimit(node.Code, node.LimitResetIndex)
	cache2.SetNodeInfo(cache2.NodeInfo{
		Code:            node.Code,
		LimitResetIndex: node.LimitResetIndex,
		LimitTotal:      node.LimitTotal,
		LimitKind:       node.LimitKind,
	})
	node_port.Arrange(db, node.Code)
	return nil
}
