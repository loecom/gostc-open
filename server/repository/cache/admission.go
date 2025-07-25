package cache

import (
	"github.com/patrickmn/go-cache"
	"server/global"
)

const (
	admission_key = "admission:"
)

type AdmissionInfo struct {
	Code        string
	WhiteEnable int
	WhiteList   []string
	WhiteMap    map[string]struct{}
}

func (admission *AdmissionInfo) ValidWhiteIp(ip string) bool {
	if admission.WhiteEnable != 1 {
		return true
	}
	if _, ok := admission.WhiteMap[ip]; ok {
		return true
	}
	return false
}

func SetAdmissionInfo(info AdmissionInfo) {
	info.WhiteMap = make(map[string]struct{})
	for _, item := range info.WhiteList {
		info.WhiteMap[item] = struct{}{}
	}
	global.Cache.SetStruct(admission_key+info.Code, info, cache.NoExpiration)
}

func GetAdmissionInfo(code string) (result AdmissionInfo) {
	_ = global.Cache.GetStruct(admission_key+code, &result)
	return result
}

func DelAdmissionInfo(code string) {
	global.Cache.Del(admission_key + code)
}
