package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

const (
	Success    = 0
	ParamError = 1
	UserExist  = 2
	SaveError  = 3
	UserEmpty  = 4
	TokenEmpty = 5
)

var TrimAllSpaceReg = regexp.MustCompile("[\\n\\t\\r\\s]")
var TrimBreakLineReg = regexp.MustCompile("[\\n\\t\\r]")

type Response struct {
	Msg string `json:"msg"`
}

type HTTPError struct {
	Code int
	Err  error
}

type CommonInfo struct {
	PluginAddr string
	PluginPort int
	User       string
	Pwd        string
}

type TokenInfo struct {
	User       string `json:"user" form:"user"`
	Token      string `json:"token" form:"token"`
	Comment    string `json:"comment" form:"comment"`
	Ports      string `json:"ports" from:"ports"`
	Domains    string `json:"domains" from:"domains"`
	Subdomains string `json:"subdomains" from:"subdomains"`
	Status     bool   `json:"status" form:"status"`
}

type TokenResponse struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Count int         `json:"count"`
	Data  []TokenInfo `json:"data"`
}

type OperationResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type TokenSearch struct {
	TokenInfo
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

type TokenUpdate struct {
	Before TokenInfo `json:"before"`
	After  TokenInfo `json:"after"`
}

type TokenRemove struct {
	Users []TokenInfo `json:"users"`
}

type TokenDisable struct {
	TokenRemove
}

type TokenEnable struct {
	TokenDisable
}

func (e *HTTPError) Error() string {
	return e.Err.Error()
}

type HandlerFunc func(ctx *gin.Context) (interface{}, error)

func (c *HandleController) MakeHandlerFunc() gin.HandlerFunc {
	return func(context *gin.Context) {
		var response plugin.Response
		var err error

		request := plugin.Request{}
		if err := context.BindJSON(&request); err != nil {
			_ = context.Error(&HTTPError{
				Code: http.StatusBadRequest,
				Err:  err,
			})
			return
		}

		jsonStr, err := json.Marshal(request.Content)
		if err != nil {
			_ = context.Error(&HTTPError{
				Code: http.StatusBadRequest,
				Err:  err,
			})
			return
		}

		if request.Op == "Login" {
			content := plugin.LoginContent{}
			err = json.Unmarshal(jsonStr, &content)
			response = c.HandleLogin(&content)
		} else if request.Op == "NewProxy" {
			content := plugin.NewProxyContent{}
			err = json.Unmarshal(jsonStr, &content)
			response = c.HandleNewProxy(&content)
		} else if request.Op == "Ping" {
			content := plugin.PingContent{}
			err = json.Unmarshal(jsonStr, &content)
			response = c.HandlePing(&content)
		} else if request.Op == "NewWorkConn" {
			content := plugin.NewWorkConnContent{}
			err = json.Unmarshal(jsonStr, &content)
			response = c.HandleNewWorkConn(&content)
		} else if request.Op == "NewUserConn" {
			content := plugin.NewUserConnContent{}
			err = json.Unmarshal(jsonStr, &content)
			response = c.HandleNewUserConn(&content)
		}

		if err != nil {
			log.Printf("handle %s error: %v", context.Request.URL.Path, err)
			var e *HTTPError
			switch {
			case errors.As(err, &e):
				context.JSON(e.Code, &Response{Msg: e.Err.Error()})
			default:
				context.JSON(http.StatusInternalServerError, &Response{Msg: err.Error()})
			}
			return
		} else {
			resStr, _ := json.Marshal(response)
			log.Printf("handle:%v , result: %v", request.Op, string(resStr))
		}

		context.JSON(http.StatusOK, response)
	}
}

func (c *HandleController) MakeManagerFunc() func(context *gin.Context) {
	return func(context *gin.Context) {
		context.HTML(http.StatusOK, "index.html", gin.H{
			"UserManage":                   ginI18n.MustGetMessage(context, "User Manage"),
			"User":                         ginI18n.MustGetMessage(context, "User"),
			"Token":                        ginI18n.MustGetMessage(context, "Token"),
			"Notes":                        ginI18n.MustGetMessage(context, "Notes"),
			"Search":                       ginI18n.MustGetMessage(context, "Search"),
			"Reset":                        ginI18n.MustGetMessage(context, "Reset"),
			"NewUser":                      ginI18n.MustGetMessage(context, "New user"),
			"RemoveUser":                   ginI18n.MustGetMessage(context, "Remove user"),
			"DisableUser":                  ginI18n.MustGetMessage(context, "Disable user"),
			"EnableUser":                   ginI18n.MustGetMessage(context, "Enable user"),
			"Remove":                       ginI18n.MustGetMessage(context, "Remove"),
			"Enable":                       ginI18n.MustGetMessage(context, "Enable"),
			"Disable":                      ginI18n.MustGetMessage(context, "Disable"),
			"PleaseInputUserAccount":       ginI18n.MustGetMessage(context, "Please input user account"),
			"PleaseInputUserToken":         ginI18n.MustGetMessage(context, "Please input user token"),
			"PleaseInputUserNotes":         ginI18n.MustGetMessage(context, "Please input user notes"),
			"AllowedPorts":                 ginI18n.MustGetMessage(context, "Allowed ports"),
			"PleaseInputAllowedPorts":      ginI18n.MustGetMessage(context, "Please input allowed ports"),
			"AllowedDomains":               ginI18n.MustGetMessage(context, "Allowed domains"),
			"PleaseInputAllowedDomains":    ginI18n.MustGetMessage(context, "Please input allowed domains"),
			"AllowedSubdomains":            ginI18n.MustGetMessage(context, "Allowed subdomains"),
			"PleaseInputAllowedSubdomains": ginI18n.MustGetMessage(context, "Please input allowed subdomains"),
			"NotLimit":                     ginI18n.MustGetMessage(context, "Not limit"),
			"None":                         ginI18n.MustGetMessage(context, "None"),
		})
	}
}

func (c *HandleController) MakeLangFunc() func(context *gin.Context) {
	return func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"User":                  ginI18n.MustGetMessage(context, "User"),
			"Token":                 ginI18n.MustGetMessage(context, "Token"),
			"Notes":                 ginI18n.MustGetMessage(context, "Notes"),
			"Status":                ginI18n.MustGetMessage(context, "Status"),
			"Operation":             ginI18n.MustGetMessage(context, "Operation"),
			"Enable":                ginI18n.MustGetMessage(context, "Enable"),
			"Disable":               ginI18n.MustGetMessage(context, "Disable"),
			"NewUser":               ginI18n.MustGetMessage(context, "New user"),
			"Confirm":               ginI18n.MustGetMessage(context, "Confirm"),
			"Cancel":                ginI18n.MustGetMessage(context, "Cancel"),
			"RemoveUser":            ginI18n.MustGetMessage(context, "Remove user"),
			"DisableUser":           ginI18n.MustGetMessage(context, "Disable user"),
			"ConfirmRemoveUser":     ginI18n.MustGetMessage(context, "Confirm to remove user"),
			"ConfirmDisableUser":    ginI18n.MustGetMessage(context, "Confirm to disable user"),
			"TakeTimeMakeEffective": ginI18n.MustGetMessage(context, "will take sometime to make effective"),
			"ConfirmEnableUser":     ginI18n.MustGetMessage(context, "Confirm to enable user"),
			"OperateSuccess":        ginI18n.MustGetMessage(context, "Operate success"),
			"OperateError":          ginI18n.MustGetMessage(context, "Operate error"),
			"OperateFailed":         ginI18n.MustGetMessage(context, "Operate failed"),
			"UserExist":             ginI18n.MustGetMessage(context, "User exist"),
			"UserEmpty":             ginI18n.MustGetMessage(context, "User cannot be empty"),
			"TokenEmpty":            ginI18n.MustGetMessage(context, "Token cannot be empty"),
			"ShouldCheckUser":       ginI18n.MustGetMessage(context, "Please check at least one user"),
			"OperationConfirm":      ginI18n.MustGetMessage(context, "Operation confirm"),
			"EmptyData":             ginI18n.MustGetMessage(context, "Empty data"),
			"AllowedPorts":          ginI18n.MustGetMessage(context, "Allowed ports"),
			"AllowedDomains":        ginI18n.MustGetMessage(context, "Allowed domains"),
			"AllowedSubdomains":     ginI18n.MustGetMessage(context, "Allowed subdomains"),
			"PortsInvalid":          ginI18n.MustGetMessage(context, "Ports is invalid"),
			"DomainsInvalid":        ginI18n.MustGetMessage(context, "Domains is invalid"),
			"SubdomainsInvalid":     ginI18n.MustGetMessage(context, "Subdomains is invalid"),
			"CommentInvalid":        ginI18n.MustGetMessage(context, "Comment is invalid"),
			"ParamError":            ginI18n.MustGetMessage(context, "Param error"),
		})
	}
}

func (c *HandleController) MakeQueryTokensFunc() func(context *gin.Context) {
	return func(context *gin.Context) {

		search := TokenSearch{}
		search.Limit = 0

		err := context.BindQuery(&search)
		if err != nil {
			return
		}

		var tokenList []TokenInfo
		for _, tokenInfo := range c.Tokens {
			tokenList = append(tokenList, tokenInfo)
		}
		sort.Slice(tokenList, func(i, j int) bool {
			return strings.Compare(tokenList[i].User, tokenList[j].User) < 0
		})

		var filtered []TokenInfo
		for _, tokenInfo := range tokenList {
			if filter(tokenInfo, search.TokenInfo) {
				filtered = append(filtered, tokenInfo)
			}
		}
		if filtered == nil {
			filtered = []TokenInfo{}
		}

		count := len(filtered)
		if search.Limit > 0 {
			start := max((search.Page-1)*search.Limit, 0)
			end := min(search.Page*search.Limit, len(filtered))
			filtered = filtered[start:end]
		}

		context.JSON(http.StatusOK, &TokenResponse{
			Code:  0,
			Msg:   "query Tokens success",
			Count: count,
			Data:  filtered,
		})
	}
}

func filter(main TokenInfo, sub TokenInfo) bool {
	replaceSpaceUser := TrimAllSpaceReg.ReplaceAllString(sub.User, "")
	if len(replaceSpaceUser) != 0 {
		if !strings.Contains(main.User, replaceSpaceUser) {
			return false
		}
	}

	replaceSpaceToken := TrimAllSpaceReg.ReplaceAllString(sub.Token, "")
	if len(replaceSpaceToken) != 0 {
		if !strings.Contains(main.Token, replaceSpaceToken) {
			return false
		}
	}

	replaceSpaceComment := TrimAllSpaceReg.ReplaceAllString(sub.Comment, "")
	if len(replaceSpaceComment) != 0 {
		if !strings.Contains(main.Comment, replaceSpaceComment) {
			return false
		}
	}
	return true
}

func (c *HandleController) MakeAddTokenFunc() func(context *gin.Context) {
	return func(context *gin.Context) {
		info := TokenInfo{
			Status: true,
		}
		response := OperationResponse{
			Success: true,
			Code:    Success,
			Message: "user add success",
		}
		err := context.BindJSON(&info)
		if err != nil {
			log.Printf("user add failed, param error : %v", err)
			response.Success = false
			response.Code = ParamError
			response.Message = "user add failed, param error "
			context.JSON(http.StatusOK, &response)
			return
		}
		if strings.TrimSpace(info.User) == "" {
			log.Printf("user add failed, user cannot be empty")
			response.Success = false
			response.Code = UserEmpty
			response.Message = fmt.Sprintf("user add failed, user cannot be empty")
			context.JSON(http.StatusOK, &response)
			return
		}
		if _, exist := c.Tokens[info.User]; exist {
			log.Printf("user add failed, user [%v] exist", info.User)
			response.Success = false
			response.Code = UserExist
			response.Message = fmt.Sprintf("user add failed, user [%s] exist ", info.User)
			context.JSON(http.StatusOK, &response)
			return
		}
		if strings.TrimSpace(info.Token) == "" {
			log.Printf("user add failed, token cannot be empty")
			response.Success = false
			response.Code = TokenEmpty
			response.Message = fmt.Sprintf("user add failed, token cannot be empty")
			context.JSON(http.StatusOK, &response)
			return
		}
		c.Tokens[info.User] = info

		usersSection, _ := c.IniFile.GetSection("users")
		key, err := usersSection.NewKey(info.User, info.Token)
		key.Comment = info.Comment

		replaceSpacePorts := TrimAllSpaceReg.ReplaceAllString(info.Ports, "")
		if len(replaceSpacePorts) != 0 {
			portsSection, _ := c.IniFile.GetSection("ports")
			key, err = portsSection.NewKey(info.User, replaceSpacePorts)
			key.Comment = fmt.Sprintf("user %s allowed ports", info.User)
		}

		replaceSpaceDomains := TrimAllSpaceReg.ReplaceAllString(info.Domains, "")
		if len(replaceSpaceDomains) != 0 {
			domainsSection, _ := c.IniFile.GetSection("domains")
			key, err = domainsSection.NewKey(info.User, replaceSpaceDomains)
			key.Comment = fmt.Sprintf("user %s allowed domains", info.User)
		}

		replaceSpaceSubdomains := TrimAllSpaceReg.ReplaceAllString(info.Subdomains, "")
		if len(replaceSpaceSubdomains) != 0 {
			subdomainsSection, _ := c.IniFile.GetSection("subdomains")
			key, err = subdomainsSection.NewKey(info.User, replaceSpaceSubdomains)
			key.Comment = fmt.Sprintf("user %s allowed subdomains", info.User)
		}

		err = c.IniFile.SaveTo(c.ConfigFile)
		if err != nil {
			log.Printf("add failed, error : %v", err)
			response.Success = false
			response.Code = SaveError
			response.Message = "user add failed"
			context.JSON(http.StatusOK, &response)
			return
		}

		context.JSON(0, &response)
	}
}

func (c *HandleController) MakeUpdateTokensFunc() func(context *gin.Context) {
	return func(context *gin.Context) {
		response := OperationResponse{
			Success: true,
			Code:    Success,
			Message: "user update success",
		}
		update := TokenUpdate{}
		err := context.BindJSON(&update)
		if err != nil {
			log.Printf("update failed, param error : %v", err)
			response.Success = false
			response.Code = ParamError
			response.Message = "user update failed, param error "
			context.JSON(http.StatusOK, &response)
			return
		}

		after := update.After
		before := update.Before

		usersSection, _ := c.IniFile.GetSection("users")
		key, err := usersSection.GetKey(before.User)
		comment := TrimBreakLineReg.ReplaceAllString(after.Comment, "")
		after.Comment = comment
		key.Comment = comment
		key.SetValue(after.Token)

		if before.Ports != after.Ports {
			portsSection, _ := c.IniFile.GetSection("ports")
			replaceSpacePorts := TrimAllSpaceReg.ReplaceAllString(after.Ports, "")
			after.Ports = replaceSpacePorts
			ports := strings.Split(replaceSpacePorts, ",")
			if len(replaceSpacePorts) != 0 {
				key, err = portsSection.NewKey(after.User, replaceSpacePorts)
				key.Comment = fmt.Sprintf("user %s allowed ports", after.User)
				c.Ports[after.User] = ports
			} else {
				portsSection.DeleteKey(after.User)
				delete(c.Ports, after.User)
			}
		}

		if before.Domains != after.Domains {
			domainsSection, _ := c.IniFile.GetSection("domains")
			replaceSpaceDomains := TrimAllSpaceReg.ReplaceAllString(after.Domains, "")
			after.Domains = replaceSpaceDomains
			domains := strings.Split(replaceSpaceDomains, ",")
			if len(replaceSpaceDomains) != 0 {
				key, err = domainsSection.NewKey(after.User, replaceSpaceDomains)
				key.Comment = fmt.Sprintf("user %s allowed domains", after.User)
				c.Domains[after.User] = domains
			} else {
				domainsSection.DeleteKey(after.User)
				delete(c.Domains, after.User)
			}
		}

		if before.Subdomains != after.Subdomains {
			subdomainsSection, _ := c.IniFile.GetSection("subdomains")
			replaceSpaceSubdomains := TrimAllSpaceReg.ReplaceAllString(after.Subdomains, "")
			after.Subdomains = replaceSpaceSubdomains
			subdomains := strings.Split(replaceSpaceSubdomains, ",")
			if len(replaceSpaceSubdomains) != 0 {
				key, err = subdomainsSection.NewKey(after.User, replaceSpaceSubdomains)
				key.Comment = fmt.Sprintf("user %s allowed subdomains", after.User)
				c.Subdomains[after.User] = subdomains
			} else {
				subdomainsSection.DeleteKey(after.User)
				delete(c.Subdomains, after.User)
			}
		}

		c.Tokens[after.User] = after

		err = c.IniFile.SaveTo(c.ConfigFile)
		if err != nil {
			log.Printf("user update failed, error : %v", err)
			response.Success = false
			response.Code = SaveError
			response.Message = "user update failed"
			context.JSON(http.StatusOK, &response)
			return
		}

		context.JSON(http.StatusOK, &response)
	}
}

func (c *HandleController) MakeRemoveTokensFunc() func(context *gin.Context) {
	return func(context *gin.Context) {
		response := OperationResponse{
			Success: true,
			Code:    Success,
			Message: "user remove success",
		}
		remove := TokenRemove{}
		err := context.BindJSON(&remove)
		if err != nil {
			log.Printf("user remove failed, param error : %v", err)
			response.Success = false
			response.Code = ParamError
			response.Message = "user remove failed, param error "
			context.JSON(http.StatusOK, &response)
			return
		}

		usersSection, _ := c.IniFile.GetSection("users")
		for _, user := range remove.Users {
			delete(c.Tokens, user.User)
			usersSection.DeleteKey(user.User)
		}

		portsSection, _ := c.IniFile.GetSection("ports")
		for _, user := range remove.Users {
			delete(c.Ports, user.User)
			portsSection.DeleteKey(user.User)
		}

		domainsSection, _ := c.IniFile.GetSection("domains")
		for _, user := range remove.Users {
			delete(c.Domains, user.User)
			domainsSection.DeleteKey(user.User)
		}

		subdomainsSection, _ := c.IniFile.GetSection("subdomains")
		for _, user := range remove.Users {
			delete(c.Subdomains, user.User)
			subdomainsSection.DeleteKey(user.User)
		}

		err = c.IniFile.SaveTo(c.ConfigFile)
		if err != nil {
			log.Printf("user remove failed, error : %v", err)
			response.Success = false
			response.Code = SaveError
			response.Message = "user remove failed"
			context.JSON(http.StatusOK, &response)
			return
		}

		context.JSON(http.StatusOK, &response)
	}
}

func (c *HandleController) MakeDisableTokensFunc() func(context *gin.Context) {
	return func(context *gin.Context) {
		response := OperationResponse{
			Success: true,
			Code:    Success,
			Message: "remove success",
		}
		disable := TokenDisable{}
		err := context.BindJSON(&disable)
		if err != nil {
			log.Printf("disable failed, param error : %v", err)
			response.Success = false
			response.Code = ParamError
			response.Message = "disable failed, param error "
			context.JSON(http.StatusOK, &response)
			return
		}

		section, _ := c.IniFile.GetSection("disabled")
		for _, user := range disable.Users {
			section.DeleteKey(user.User)
			token := c.Tokens[user.User]
			token.Status = false
			c.Tokens[user.User] = token
			key, err := section.NewKey(user.User, "disable")
			if err != nil {
				log.Printf("disable failed, error : %v", err)
				response.Success = false
				response.Code = SaveError
				response.Message = "disable failed"
				context.JSON(http.StatusOK, &response)
				return
			}
			key.Comment = fmt.Sprintf("disable user '%s'", user.User)
		}

		err = c.IniFile.SaveTo(c.ConfigFile)
		if err != nil {
			log.Printf("disable failed, error : %v", err)
			response.Success = false
			response.Code = SaveError
			response.Message = "disable failed"
			context.JSON(http.StatusOK, &response)
			return
		}

		context.JSON(http.StatusOK, &response)
	}
}

func (c *HandleController) MakeEnableTokensFunc() func(context *gin.Context) {
	return func(context *gin.Context) {
		response := OperationResponse{
			Success: true,
			Code:    Success,
			Message: "remove success",
		}
		enable := TokenEnable{}
		err := context.BindJSON(&enable)
		if err != nil {
			log.Printf("enable failed, param error : %v", err)
			response.Success = false
			response.Code = ParamError
			response.Message = "enable failed, param error "
			context.JSON(http.StatusOK, &response)
			return
		}

		section, _ := c.IniFile.GetSection("disabled")
		for _, user := range enable.Users {
			section.DeleteKey(user.User)
			token := c.Tokens[user.User]
			token.Status = true
			c.Tokens[user.User] = token
		}

		err = c.IniFile.SaveTo(c.ConfigFile)
		if err != nil {
			log.Printf("enable failed, error : %v", err)
			response.Success = false
			response.Code = SaveError
			response.Message = "enable failed"
			context.JSON(http.StatusOK, &response)
			return
		}

		context.JSON(http.StatusOK, &response)
	}
}
