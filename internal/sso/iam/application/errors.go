package application

import (
	"net/http"

	"hongzewei.sso/internal/shared/errcode"
)

// iam 上下文错误码(2000 段)。
// 安全约定:登录失败不区分「账号不存在 / 密码错误」,统一返回同一文案,防止用户枚举。
var (
	SSOIAMLoginFailed             = errcode.Register(2000, http.StatusUnauthorized, "账号或密码错误")
	SSOIAMAccountDisabled         = errcode.Register(2001, http.StatusForbidden, "账号已被禁用")
	SSOIAMAccountLocked           = errcode.Register(2002, http.StatusForbidden, "账号已被锁定,请联系管理员")
	SSOIAMLoginChallengeInvalid   = errcode.Register(2003, http.StatusBadRequest, "登录请求无效或已过期,请重新从应用发起登录")
	SSOIAMConsentChallengeMissing = errcode.Register(2004, http.StatusBadRequest, "缺少 consent_challenge 参数")
	SSOIAMConsentUserInvalid      = errcode.Register(2005, http.StatusBadRequest, "授权用户标识无效")
	SSOIAMHydraUnavailable        = errcode.Register(2006, http.StatusBadGateway, "认证服务暂时不可用,请稍后重试")
	SSOIAMUserNotFound            = errcode.Register(2007, http.StatusNotFound, "用户不存在")
	SSOIAMOldPasswordWrong        = errcode.Register(2008, http.StatusBadRequest, "原密码错误")
	SSOIAMLogoutChallengeMissing  = errcode.Register(2009, http.StatusBadRequest, "缺少 logout_challenge 参数")
	SSOIAMPasswordTooShort        = errcode.Register(2010, http.StatusBadRequest, "密码长度不能少于 6 位")
	SSOIAMUsernameExists          = errcode.Register(2011, http.StatusConflict, "用户名已存在")
	SSOIAMUserCreateFailed        = errcode.Register(2012, http.StatusInternalServerError, "创建用户失败")
	SSOIAMUserUpdateFailed        = errcode.Register(2013, http.StatusInternalServerError, "更新用户失败")
	SSOIAMCannotDeleteSelf        = errcode.Register(2014, http.StatusForbidden, "不能删除自己")
)
