package main

import (
	io2 "./io"
	"./ticket"
	"./ticket/model"
	"encoding/json"
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
	Version = "2.0.1"
)

var (
	cookieNameForSessionID ="sessionid"
	sess = sessions.New(sessions.Config{Cookie:cookieNameForSessionID})
	counterLock sync.Mutex
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ticketTimeout := flag.Int("timeout", 3600, "下载链接超时时间(秒)")
	isDebug := flag.Bool("debug", false, "是否启用log调试模式(true或false), 默认为false")
	listen := flag.String("listen", "[::]:8099", "设置监听地址")

	flag.Parse()

	//设置下载链接超时时间
	tm := ticket.NewTicketManager(*ticketTimeout)

	app := iris.New()
	app.Logger().Info("下载链接超时时间 -> ", *ticketTimeout)

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

			//读取POST JSON表单参数
			params := make(map[string]interface{})
			if err := context.ReadJSON(&params); err != nil {
				app.Logger().Error(err)
			}
			app.Logger().Info(params)

			//开始操作session
			session := sess.Start(context)
			//获取session中保存的验证码captchaKey
			captchaKey := session.GetString("captchakey")

			if len(captchaKey)==0 {
				showJSON(context, 20001, "请先请求验证码图像", nil)
				return
			}

			if params["verify"]==nil || len(params["verify"].(string))==0 {
				showJSON(context, 20002, "缺少验证码参数", nil)
				return
			}
			//获取POST请求提交过来的验证码值参数
			verifyCode := params["verify"].(string)

			app.Logger().Debug("capid->", captchaKey, "  value->", verifyCode)

			//判断验证码是否正确
			if !base64Captcha.VerifyCaptcha(captchaKey, verifyCode) {
				showJSON(context, 20000, "验证码错误", nil)
				return
			}

			//判断url参数是否设置
			if params["url"] == nil {
				showJSON(context, 10000, "缺少URL参数", nil)
				return
			}
			paramUrl := params["url"].(string)

			//如果urldecode==true则进行URL解码
			if params["urldecode"] != nil && params["urldecode"] == true {
				unescapeUrl, err := url.QueryUnescape(paramUrl)
				if err != nil {
					showJSON(context, 10010, "url参数解码失败  详细信息->"+err.Error(), nil)
					return
				}
				app.Logger().Debug(unescapeUrl)
				paramUrl = unescapeUrl
			}
			if params["headers"] != nil {
				app.Logger().Info("Set session data: Custom Headers -> ", params["headers"])
			}

			var filename, cookie string
			var headers []map[string]interface{}
			if params["filename"] != nil {
				filename = params["filename"].(string)
			}

			//如果headers(自定义协议头)不为空则进行json解码
			if params["headers"] != nil {
				//headers = params["headers"].(map[string]string)
				log.Print([]byte(params["headers"].(string)), params["headers"].(string))
				err := json.Unmarshal([]byte(params["headers"].(string)), &headers)
				if err != nil {
					showJSON(context, 10011, "header json参数解码失败  详细信息->"+err.Error(), nil)
					return
				}
			}
			if params["cookie"] != nil {
				cookie = params["cookie"].(string)
			}

			t := tm.GenerateTicket(&model.TicketData {
				URL:              paramUrl,
				FileName:         filename,
				Headers:          headers,
				Cookie:           cookie,
			})

			showJSON(context, 0, "ok", func(params map[string]interface{}) {
				params["token"] = t
				params["data"] = tm.GetTicket(t)
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

	//下载文件接口
	api.Get("/download/{token:string}", func(context context.Context) {
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

		//创建HTTP请求
		req, err := http.NewRequest("GET", t.URL, nil)
		if err != nil {
			context.StatusCode(502)
			showErrorPage(context, err.Error())
			return
		}

		//输出客户端HTTP Request头字段
		for k, v := range context.Request().Header {
			req.Header.Set(k, v[0])
			app.Logger().Debug("Request Header: ", k, " -> ", v[0])
		}

		//处理自定义Headers，加入到Request Header
		if t.Headers != nil {

			for i:=0; i< len(t.Headers); i++ {
				for k, v := range t.Headers[i] {
					req.Header.Set(k, v.(string))
					app.Logger().Debug("Custom Request Header: ", k, " -> ", v)
				}
			}

			/*for k, v := range t.Headers {
				req.Header.Set(k, v[0].(string))
				app.Logger().Debug("Custom Request Header: ", k, " -> ", v)
			}*/
		}

		//如果设置了自定义Cookie则加入到Request Header
		if len(t.Cookie) > 0 {
			req.Header.Set("Cookie", t.Cookie)
		}

		//发出HTTP请求
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			context.StatusCode(502)
			showErrorPage(context, err.Error())
			return
		}

		//转发HTTP状态码
		context.StatusCode(resp.StatusCode)
		app.Logger().Debug("状态码->", resp.StatusCode)

		//转发Response Header
		for k, v := range resp.Header {
			if k != "Server" {
				context.Header(k, v[0])
			}
			app.Logger().Debug("Response Header: ", k, " -> ", v[0])
		}
		//自定义文件名
		if len(t.FileName) > 0 {
			context.Header("content-disposition", "attachment; filename=\""+t.FileName+"\"")
		}

		app.Logger().Debug("token->", token, ", 开始转发数据流")
		//len, err := io.Copy(context, resp.Body)
		len, err := io2.Copy(context, resp.Body)
		if err != nil {
			app.Logger().Debug("token->", token, ", IO错误, 详细信息->", err.Error())
		}

		app.Logger().Debug("token->", token, ", 数据流关闭, len->", len)

		counterLock.Lock()
		t.DownloadCounter++
		counterLock.Unlock()
	})

	//api.Get("/download/{token:string}/{filename:string}", download)
	runtime.GOMAXPROCS(runtime.NumCPU())

	app.StaticWeb("/", "./www")

	// 自定义错误页面
	app.RegisterView(iris.HTML("./views", ".html"))
	app.Run(iris.Addr(*listen))
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
