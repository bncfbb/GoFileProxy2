package main

import (
	"./controller"
	io2 "./io"
	"./model"
	"flag"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
	"github.com/mojocn/base64Captcha"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
)

const (
	Version = "2.0.3"
)

var (
	cookieNameForSessionID ="sessionid"
	sess = sessions.New(sessions.Config{Cookie:cookieNameForSessionID})
	counterLock sync.Mutex

	buffsize int64

	domain string
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ticketTimeout := flag.Int("timeout", 3600, "下载链接超时时间(秒)")
	isDebug := flag.Bool("debug", false, "是否启用log调试模式(true或false), 默认为false")
	buffsize := *flag.Int64("buffersize", 8192, "缓冲区大小")
	listen := flag.String("listen", "[::]:8099", "设置监听地址")

	autoHttps := flag.Bool("auto-https", false, "是否启用自动获取HTTPS SSL证书(true或false), 默认为false")
	domain := flag.String("domain", "", "站点域名, 启用自动获取HTTPS SSL证书时需要填写")
	email := flag.String("email", "", "邮箱, 启用自动获取HTTPS SSL证书时需要填写")

	https := flag.Bool("https", false, "是否启用手动https(true或false), 默认为false, 启用时需要填写cert和key参数")
	cert := flag.String("cert", "", "SSL证书, 启用手动https需要")
	key := flag.String("key", "", "SSL 私钥, 启用手动https需要")

	flag.Parse()

	//设置下载链接超时时间
	tm := controller.NewTicketManager(*ticketTimeout)

	app := iris.New()
	app.Logger().Info("下载链接超时时间 -> ", *ticketTimeout)

	app.Logger().Info("数据流转发缓冲区大小 -> ", buffsize)

	//app.Logger().SetLevel("debug")
	if *isDebug {
		app.Logger().SetLevel("debug")
		app.Logger().Info("开启log debug模式")
	}
	app.Logger().Info("监听地址 -> ", *listen)

	app.Use(func(context context.Context) {
		context.Header("Server", "GoProxy/" + Version)
		context.Next()
	})
	
	api := app.Party("/api")
	{
		//验证码图片接口
		api.Get("/captcha/image", func(context context.Context) {
			session := sess.Start(context)
			//config struct for Character
			var configC = base64Captcha.ConfigCharacter{
				Height:             40,
				Width:              160,
				//const CaptchaModeNumber:数字,CaptchaModeAlphabet:字母,CaptchaModeArithmetic:算术,CaptchaModeNumberAlphabet:数字字母混合.
				Mode:               base64Captcha.CaptchaModeNumberAlphabet,
				ComplexOfNoiseText: base64Captcha.CaptchaComplexLower,
				ComplexOfNoiseDot:  base64Captcha.CaptchaComplexLower,
				IsShowHollowLine:   false,
				IsShowNoiseDot:     false,
				IsShowNoiseText:    false,
				IsShowSlimeLine:    false,
				IsShowSineLine:     false,
				CaptchaLen:         4,
			}
			//创建字符公式验证码.
			//GenerateCaptcha 第一个参数为空字符串,包会自动在服务器一个随机种子给你产生随机uiid.
			idKey, cap := base64Captcha.GenerateCaptcha("", configC)

			app.Logger().Debug(idKey)
			session.Set("captchakey", idKey)
			cap.WriteTo(context)
		})

		//用于测试验证码接口
		api.Get("/captcha/verify/{code:string}", func(context context.Context) {
			session := sess.Start(context)
			captchaKey := session.GetString("captchakey")
			code := context.Params().Get("code")

			app.Logger().Debug("capid->", captchaKey, "  value->", code)
			if base64Captcha.VerifyCaptcha(captchaKey, code) {
				showJSON(context, 0, "ok", nil)
				return
			}
			showJSON(context, 20000, "验证码错误", nil)
		})

		//生成token获取下载地址接口
		api.Post("/token/generate", func(context context.Context) {
			var filename, cookie string
			var headers []map[string]interface{}

			//读取POST JSON表单参数
			params, err := controller.NewParamReader(context)
			if err != nil {
				showJSON(context, 20004, "POST参数解析失败", nil)
				return
			}

			//开始操作session
			session := sess.Start(context)
			//获取session中保存的验证码captchaKey
			captchaKey := session.GetString("captchakey")

			if len(captchaKey)==0 {
				showJSON(context, 20001, "请先请求验证码图像", nil)
				return
			}


			//获取POST请求提交过来的验证码值参数
			verifyCode, success := params.GetJsonParamString("verify")
			if !success {
				showJSON(context, 20002, "缺少验证码参数", nil)
				return
			}

			app.Logger().Debug("capid->", captchaKey, "  value->", verifyCode)

			//判断验证码是否正确
			if !base64Captcha.VerifyCaptcha(captchaKey, verifyCode) {
				showJSON(context, 20000, "验证码错误", nil)
				return
			}

			paramUrl, success := params.GetJsonParamString("url")
			//判断url参数是否设置
			if !success {
				showJSON(context, 10000, "缺少URL参数", nil)
				return
			}

			//如果urldecode==true则进行URL解码
			if params.GetJsonParamBool("urldecode") {
				unescapeUrl, err := url.QueryUnescape(paramUrl)
				if err != nil {
					showJSON(context, 10010, "url参数解码失败  详细信息->"+err.Error(), nil)
					return
				}
				app.Logger().Debug(unescapeUrl)
				paramUrl = unescapeUrl
			}

			temp, success := params.GetJsonParamString("filename")
			if success {
				filename = temp
			}

			//如果headers(自定义协议头)不为空则进行json解码
			if params.IsValidParam("headers") {
				h, err := params.GetJsonParamToMap("headers")
				if err != nil {
					showJSON(context, 10011, "header json参数解码失败  详细信息->"+err.Error(), nil)
					return
				}
				app.Logger().Info("Set session data: Custom Headers -> ", h)
				headers = h
			}

			//自定义cookie
			temp, success = params.GetJsonParamString("cookie")
			if success {
				cookie = temp
			}

			//生成计时ticket
			t := tm.GenerateTicket(&model.TicketData {
				URL:              paramUrl,
				FileName:         filename,
				Headers:          headers,
				Cookie:           cookie,
			})

			tObj := tm.GetTicket(t)
			showJSON(context, 0, "ok", func(params map[string]interface{}) {
				params["token"] = t
				params["data"] = &map[string]interface{} {
					"source": tObj.URL,
					"filename": tObj.FileName,
					"headers": tObj.Headers,
					"cookie": tObj.Cookie,
					"generate_timestamp": tObj.StartTimeStamp,
					"expire_timestamp": tObj.ExpireTimeStamp,
				}
			})

		})

		//获取token详细信息接口
		api.Get("/token/info/{token:string}", func(context context.Context) {
			//session := sess.NewSessionManager(context)
			token := context.Params().Get("token")

			t := tm.GetTicket(token)
			if t == nil {
				showJSON(context, 10009, "Token无效或已过期", nil)
				return
			}

			showJSON(context, 0, "ok", func(params map[string]interface{}) {
				params["data"] = t
			})
		})

		//Get Version API
		api.Get("/ver", func(context context.Context) {
			showJSON(context, 0, "ok", func(mmap map[string]interface{}) {
				mmap["version"] = Version
			})
		})
	}

	//数据流转发接口
	api.Get("/stream/{token:string}", func(context context.Context) {
		token := context.Params().Get("token")
		if len(token) == 0 {
			context.StatusCode(400)
			showErrorPage(context, "缺少Token参数")
			return
		}
		t := tm.GetTicket(token)
		if t == nil {
			context.StatusCode(404)
			showErrorPage(context, "Token无效或已过期")
			return
		}

		app.Logger().Info("请求文件 -> ", token)

		//创建HTTP请求转发器
		forwarder, err := controller.NewRequestForwarder(app, context, t.URL)
		if err != nil {
			context.StatusCode(502)
			showErrorPage(context, err.Error())
			return
		}

		//转发用户HTTP Request头字段到目标网站
		forwarder.HandleRequestHeader()

		//如果设置了自定义Headers, 则额外转发用户设置的自定义Headers字段
		if t.Headers != nil {
			forwarder.SetCustomRequestHeaderMap(t.Headers)
		}

		//如果设置了自定义Cookie则加入到Request Header
		if len(t.Cookie) > 0 {
			forwarder.SetCustomRequestHeader("Cookie", t.Cookie)
		}

		//发出HTTP请求
		if err := forwarder.Do(); err != nil {
			context.StatusCode(502)
			showErrorPage(context, err.Error())
			return
		}

		//转发HTTP状态码
		context.StatusCode(forwarder.GetStatusCode())
		app.Logger().Debug("状态码->", forwarder.GetStatusCode())

		//转发Response Header
		forwarder.HandleResponseHeader()

		//自定义文件名
		if len(t.FileName) > 0 {
			context.Header("content-disposition", "attachment; filename=\""+t.FileName+"\"")
		}

		app.Logger().Debug("token->", token, ", 开始转发数据流")

		l, err := io2.Copy(context, forwarder.GetBody(), buffsize)
		if err != nil {
			app.Logger().Debug("token->", token, ", IO错误, 详细信息->", err.Error())
		}

		app.Logger().Debug("token->", token, ", 数据流关闭, len->", l)

		counterLock.Lock()
		t.DownloadCounter++
		counterLock.Unlock()
	})

	//api.Get("/download/{token:string}/{filename:string}", download)
	runtime.GOMAXPROCS(runtime.NumCPU())

	app.HandleDir("/", "./www")

	// 自定义错误页面
	app.RegisterView(iris.HTML("./views", ".html"))

	if *autoHttps {
		if len(*domain)==0 {
			app.Logger().Fatal("启用auto https需要填写domain")
			return
		}
		if len(*email)==0 {
			app.Logger().Fatal("启用auto https需要填写email")
			return
		}
		app.Run(iris.AutoTLS(*listen, *domain, *email))
	} else {
		if *https {
			if len(*cert)==0 {
				app.Logger().Fatal("启用https需要填写cert")
				return
			}
			if len(*key)==0 {
				app.Logger().Fatal("启用https需要填写key")
				return
			}
		}
		app.Run(iris.TLS(*listen, *cert, *key))
	}
}

func showErrorPage(context context.Context, message string) {
	code := context.GetStatusCode()
	context.ViewData("code", strconv.Itoa(code) + " " + http.StatusText(code))
	context.ViewData("msg", message)
	context.ViewData("server", "GoProxy/" + Version)
	context.View("error.html")
}

func showJSON(context context.Context, code int, message string, cbk func (map[string]interface{}) ) {
	data := make(map[string]interface{})
	data["code"] = code
	data["message"] = message
	if cbk != nil {
		cbk(data)
	}
	context.JSON(data)
}

