package database

import (
	"bytes"
	"crypto/md5"
	"eduhacks2020/Go/api/users"
	"eduhacks2020/Go/protobuf"
	"eduhacks2020/Go/render"
	"encoding/hex"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"net/http"
)

// 定义路由
const (
	APILogin = "/api/login" //用户登录的接口
)

// 定义一些常量错误
const (
	signInvalid = "data has been tampered with invalid sign"
)

// Router 创建类型指定 Find 方法
type Router struct {
}

// ProtoParam 这里包含了 websocket 中的 sessionId 请求体和响应体
type ProtoParam struct {
	Request   *protobuf.Request
	Response  *protobuf.Response
	SessionID string
	DB        *gorm.DB
	Redis     *redis.Client
}

type fun func(*ProtoParam)

// Call 定义一个接口
type Call interface {
	call(*ProtoParam)
}

func (f fun) call(param *ProtoParam) {
	f(param)
}

// Find websocket 处理路由的主要方法
func (r *Router) Find(p *ProtoParam, f func(param *ProtoParam)) {
	handlerFind(p, fun(f))
}

func handlerFind(p *ProtoParam, c Call) {
	c.call(p)
}

// 用于计算签名
func calcSign(timestamp string, data []byte) string {
	var buffer bytes.Buffer
	buffer.Write([]byte(timestamp))
	buffer.Write(data)
	h := md5.New()
	h.Write(buffer.Bytes())
	cipherStr := h.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

// 返回校验的结果
func verifySign(salt string, submitSign, data []byte) bool {
	submit := string(submitSign)
	calc := calcSign(salt, data)
	return submit == calc
}

// Handler 判断 websocket 传递的路由然后开始处理
func Handler(p *ProtoParam) {
	switch p.Request.Path {
	case APILogin:
		login := users.LoginParam{}
		err := json.Unmarshal(p.Request.Data, &login)
		if !verifySign(login.Salt, p.Request.Sign, p.Request.Data) {
			p.Response.Msg = signInvalid
			p.Response.Html.Code = render.GetLayer(0, render.Sad, "Error", signInvalid)
			return
		}
		if err != nil {
			p.Response.Msg = err.Error()
			p.Response.Html.Code = render.GetLayer(0, render.Incorrect, "Error", err.Error())
			return
		}
		data, errMsg, err := login.Exec(p.DB, p.Redis, p.SessionID)
		p.Response.Html.Code = render.GetLayer(0, render.Sad, "Login", errMsg)
		if err == nil {
			p.Response.Code = http.StatusOK
			p.Response.Html.Code = render.GetLayer(0, render.Smile, "Login", errMsg)
		}
		p.Response.Data = data
		p.Response.Msg = errMsg
	}
}