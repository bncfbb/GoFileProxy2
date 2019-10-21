package sess

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
)

var (
	cookieNameForSessionID = "filesession"
	sess                   = sessions.New(sessions.Config{Cookie: cookieNameForSessionID, AllowReclaim: true})
)

type SessManager struct {
	context context.Context
	session *sessions.Session
}

func (self *SessManager) SetUrl(url string) {
	self.session.Set("url", url)
}

func (self *SessManager) SetCustomHeaders(headers map[string]string) {
	self.session.Set("custom_headers", headers)
}

func (self *SessManager) GetCustomHeaders() string {
	return self.session.GetString("custom_headers")
}

func (self *SessManager) GetUrl() string {
	return self.session.GetString("url")
}

func NewSessionManager(context context.Context) (*SessManager) {
	return &SessManager{
		context: context,
		session: sess.Start(context),
	}
}